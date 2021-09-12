package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
)

type Record struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	RecordType string  `json:"record_type"`
	Id         int     `json:"id"`
}

type myJSON struct {
	Records          []Record `json:"records"`
	Totals           float64  `json:"totals"`
	TotalLiabilities float64  `json:"total_liabilities"`
	TotalAssets      float64  `json:"total_assets"`
}

func (rec Record) String() string {
	return fmt.Sprintf("{%s, %s, %d, %f}", rec.Name, rec.RecordType, rec.Id, rec.Amount)
}

func saveRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.DefaultQuery("name", "")
		recType := c.DefaultQuery("type", "")
		amount := c.DefaultQuery("id", "") //wanted to use zero, wasn't allowed
		if name == "" || recType == "" || amount == "" {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: record values missing"))
			return
		}
		if _, err := db.Exec("INSERT INTO records VALUES ($1,$2,$3)", name, recType, amount); err != nil { //////////////
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
			return
		}

		//var buffer bytes.Buffer
		//for i := 0; i < r; i++ {
		//	buffer.WriteString("Hello from Go!\n")
		//}
		c.String(http.StatusOK, "")
		//refresh page, here or front end???, or just send and load data, either one?
		//c.Redirect()
	}
}

func deleteRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.DefaultQuery("id", "")
		if id == "" {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: no record id"))
			return
		}

		if _, err := db.Exec("DELETE FROM records WHERE id = $1", id); err != nil { /// sql escaping?
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
			return
		}

		//var buffer bytes.Buffer
		c.String(http.StatusOK, "")
		//refresh page!!!!!!!!!!!!!!!!!!!!
	}
}

func getRecords(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			total            float64
			totalAssets      float64
			totalLiabilities float64
			records          []Record
		)

		//why don't I do this in main?, then the get is just a get
		if _, err := db.Exec("CREATE TABLE IF NOT EXISTS records (id serial PRIMARY KEY,name text, recType varchar(10), amount NUMERIC(20,6))"); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		rows, err := db.Query("SELECT * FROM records")
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error reading ticks: %q", err))
			return
		}

		defer rows.Close()
		//println(*rows)
		for rows.Next() {
			var (
				amount  float64
				name    string
				recType string
				id      int
			)
			if err := rows.Scan(&id, &amount, &name, &recType); err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error scanning ticks: %q", err))
				return
			}
			record := &Record{
				Name:   name,
				Amount: amount,
				//Amount: fmt.Sprintf("%f",amount),
				//:  fmt.Sprintf("%d",id),
				Id:         id,
				RecordType: recType,
			}

			total += amount //type conversion
			if recType == "ASSET" {
				totalAssets += amount
			}
			if recType == "LIABILITY" {
				totalLiabilities += amount
			}
			records = append(records, *record)
			// we can use the json.Marhal function to
			// encode the pigeon variable to a JSON string
			//data, _ := json.Marshal(record)
			//fmt.Println(string(data))
			//check error
			//c.String(http.StatusOK, fmt.Sprintf("Read from DB: %s\n", tick.String()))
		}
		jsonStruct := &myJSON{
			Records:          records,
			Totals:           total,
			TotalAssets:      totalAssets,
			TotalLiabilities: totalLiabilities,
		}
		//data, _ := json.Marshal(jsonStruct) //need this?
		//fmt.Println(string(jsonStruct.Records))
		//error check
		c.JSON(http.StatusOK, jsonStruct) //??expecting struct?
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	//tStr := os.Getenv("REPEAT")
	//repeat, err := strconv.Atoi(tStr)
	//if err != nil {
	//	log.Printf("Error converting $REPEAT to an int: %q - Using default\n", err)
	//	repeat = 5
	//}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	//router.GET("/mark", func(c *gin.Context) {
	//	c.String(http.StatusOK, string(blackfriday.Run([]byte("**hi!**"))))//what's going on here?
	//})

	//router.GET("/repeat", repeatHandler(repeat))///????

	router.GET("/records", getRecords(db))

	router.POST("/record", saveRecord(db)) /////////make sure i'm using the right verbs

	router.DELETE("/record/:id", deleteRecord(db))

	router.Run(":" + port)
	//enum or whatever for asset liability
}
