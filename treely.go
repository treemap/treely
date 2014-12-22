package main

import (
	"fmt"
	gocache "github.com/abhiyerra/gowebcommons/cache"
	render "github.com/abhiyerra/gowebcommons/render"
	"github.com/coreos/go-etcd/etcd"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	DatabaseUrlKey = "/treemap/database_url"
)

var (
	db    gorm.DB
	cache gocache.Cache
)

type Tree struct {
	Id         int64    `json:"id"`
	CommonName string   `json:"common_name"`
	LatinName  string   `json:"latin_name"`
	GeomData   []string `json:"geom",sql:"-"`
	Area       float64  `json:"area",sql:"-"`
	Center     string   `json:"center",sql:"-"`
}

func (t *Tree) GetGeodata() {
	rows, err := db.Table("tree_geoms").Select("ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom2").Where("latin_name = ?", t.LatinName).Rows()
	if err != nil {
		log.Println(err)
	}

	for rows.Next() {
		var geodata string
		rows.Scan(&geodata)
		t.GeomData = append(t.GeomData, geodata)
	}
}

func (t *Tree) GetArea() {
	var a struct {
		Area float64
	}
	db.Table("tree_geoms").Select("SUM(ST_Area(ST_Transform(geom, 900913))) as area").Where("latin_name = ?", t.LatinName).Scan(&a)

	t.Area = a.Area * 0.000189394 * 0.000189394 // Get the miles
	log.Println("Area:", t.Area)
}

func (t *Tree) GetCenter() {
	var a struct {
		Center string
	}
	db.Table("tree_geoms").Select("ST_AsGeoJSON(ST_Centroid(geom)) as center").Where("latin_name = ?", t.LatinName).Scan(&a)

	t.Center = a.Center
	log.Println("Center:", t.Center)
}

func showTreesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	treeId, _ := strconv.ParseInt(vars["treeId"], 10, 64)

	tree := cache.Get("tree/"+vars["treeId"], func() interface{} {
		tree := Tree{Id: int64(treeId)}
		db.First(&tree)
		tree.GetGeodata()
		tree.GetArea()
		tree.GetCenter()

		return tree
	})

	render.RenderJson(w, tree)
}

func nearbyTreesHandler(w http.ResponseWriter, r *http.Request) {
	var trees []Tree

	longitude := r.URL.Query().Get("long")
	latitude := r.URL.Query().Get("lat")
	log.Println("Long:", longitude, "Lat:", latitude)

	err := db.Model(Tree{}).Select("distinct trees.id, trees.latin_name, trees.common_name").
		Joins(fmt.Sprintf("INNER JOIN tree_geoms ON tree_geoms.latin_name = trees.latin_name AND ST_DWithin(ST_GeomFromText('POINT(%s %s)' , 4326)::geography, tree_geoms.geom, 160934 , true)", longitude, latitude)).
		Order("trees.latin_name asc").Scan(&trees)

	if err != nil {
		log.Println(err)
	}

	render.RenderJson(w, trees)
}

func treesHandler(w http.ResponseWriter, r *http.Request) {
	trees := cache.Get("trees", func() interface{} {
		var trees []Tree

		err := db.Model(Tree{}).Select("id, latin_name, common_name").Scan(&trees)
		if err != nil {
			log.Println(err)
		}

		return trees
	})

	render.RenderJson(w, trees)
}

type NationalPark struct {
	UnitName string `json:"name"`
	UnitCode string `json:"code"`
	GeomData string `json:"geom"`
}

func nearbyParksHandler(w http.ResponseWriter, r *http.Request) {
	var parks []NationalPark

	longitude := r.URL.Query().Get("long")
	latitude := r.URL.Query().Get("lat")
	log.Println("Long:", longitude, "Lat:", latitude)

	err := db.Model(NationalPark{}).
		Select("ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom_data, unit_name, unit_code").
		Where(fmt.Sprintf("ST_DWithin(ST_GeomFromText('POINT(%s %s)' , 4326)::geography, geom, 160934, true)", longitude, latitude)). // Within 100 miles -> 160934 meters
		Scan(&parks)
	if err != nil {
		log.Println(err)
	}

	render.RenderJson(w, parks)
}

func parksHandler(w http.ResponseWriter, r *http.Request) {
	parks := cache.Get("parks", func() interface{} {
		var parks []NationalPark
		db.Model(NationalPark{}).Select("ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom_data, unit_name, unit_code").Scan(&parks)
		return parks
	})

	render.RenderJson(w, parks)
}

type Hydrology struct {
	Name     string `json:"name"`
	GeomData string `json:"geom"`
}

func nearbyHydrologyHandler(hydroType string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var hydrology []Hydrology

		longitude := r.URL.Query().Get("long")
		latitude := r.URL.Query().Get("lat")
		log.Println("Hydro Type:", hydroType, "Long:", longitude, "Lat:", latitude)

		err := db.Table(hydroType).
			Select("ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom_data, name").
			Where(fmt.Sprintf("ST_DWithin(ST_GeomFromText('POINT(%s %s)' , 4326)::geography, geom, 160934, true)", longitude, latitude)). // Within 100 miles -> 160934 meters
			Scan(&hydrology)
		if err != nil {
			log.Println(err)
		}

		render.RenderJson(w, hydrology)
	}
}

func hydrologyHandler(hydroType string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hydrology := cache.Get(hydroType, func() interface{} {
			var hydrology []Hydrology
			db.Table(hydroType).Select("ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom_data, name").Scan(&hydrology)

			return hydrology
		})

		render.RenderJson(w, hydrology)
	}
}

type Zipcode struct {
	Number   string `json:"number"`
	GeomData string `json:"geom"`
	Center   string `json:"center"`
}

func showZipCodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zipcode := vars["zipcode"]

	z := cache.Get("zipcode/"+zipcode, func() interface{} {
		z := Zipcode{Number: zipcode}
		db.Select("geoid10 as number, ST_AsGeoJSON(ST_Centroid(geom)) as center, ST_AsGeoJSON(ST_CollectionExtract(geom, 3)) as geom_data").Where("geoid10 = ?", zipcode).First(&z)

		return z
	})

	render.RenderJson(w, z)
}

func dbConnect(databaseUrl string) {
	log.Println("Connecting to database:", databaseUrl)
	var err error
	db, err = gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Println(err)
	}
	db.LogMode(true)
}

func init() {
	etcdHosts := os.Getenv("ETCD_HOSTS")
	if etcdHosts == "" {
		etcdHosts = "http://127.0.0.1:4001"
	}

	etcdClient := etcd.NewClient([]string{etcdHosts})

	resp, err := etcdClient.Get(DatabaseUrlKey, false, false)
	if err != nil {
		panic(err)
	}

	databaseUrl := resp.Node.Value
	dbConnect(databaseUrl)

	cache = gocache.New()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/trees/nearby", nearbyTreesHandler).Methods("GET")
	r.HandleFunc("/trees/{treeId}", showTreesHandler).Methods("GET")
	r.HandleFunc("/trees", treesHandler).Methods("GET")

	r.HandleFunc("/parks/nearby", nearbyParksHandler).Methods("GET")
	r.HandleFunc("/parks", parksHandler).Methods("GET")

	r.HandleFunc("/lakes/nearby", nearbyHydrologyHandler("lakes")).Methods("GET")
	r.HandleFunc("/lakes", hydrologyHandler("lakes")).Methods("GET")

	r.HandleFunc("/rivers/nearby", nearbyHydrologyHandler("rivers")).Methods("GET")
	r.HandleFunc("/rivers", hydrologyHandler("rivers")).Methods("GET")

	r.HandleFunc("/zipcode/{zipcode}", showZipCodeHandler).Methods("GET")

	http.Handle("/", r)
	http.ListenAndServe(":3001", nil)
}
