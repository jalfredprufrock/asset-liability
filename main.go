package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"io/ioutil"
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

//func (rec Record) String() string { ///
//	return fmt.Sprintf("{%s, %s, %d, %f}", rec.Name, rec.RecordType, rec.Id, rec.Amount)
//}

func saveRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			panic(err)
		}
		//log.Println(string(body))
		var newRecord Record
		err = json.Unmarshal(body, &newRecord)
		if err != nil {
			panic(err)
		}
		//log.Println(newRecord)
		if newRecord.RecordType == "" {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: record type missing"))
			return
		}
		if _, err := db.Exec("INSERT INTO records VALUES (DEFAULT,$1,$2,$3)", newRecord.Name, newRecord.RecordType, newRecord.Amount); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
			return
		}

		c.String(http.StatusCreated, "")
	}
}

func deleteRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Params.ByName("id")
		if id == "" {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: no record id"))
			return
		}

		if _, err := db.Exec("DELETE FROM records WHERE id = $1", id); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
			return
		}

		c.String(http.StatusOK, "")
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

		for rows.Next() {
			var (
				amount  float64
				name    string
				recType string
				id      int
			)
			if err := rows.Scan(&id, &name, &recType, &amount); err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error reading records db: %q", err))
				return
			}
			record := &Record{
				Name:       name,
				Amount:     amount,
				Id:         id,
				RecordType: recType,
			}

			//type conversion
			if recType == "Asset" {
				totalAssets += amount
				total += amount
			}
			if recType == "Liability" {
				totalLiabilities += amount
				total -= amount
			}
			records = append(records, *record)

		}
		jsonStruct := &myJSON{
			Records:          records,
			Totals:           total,
			TotalAssets:      totalAssets,
			TotalLiabilities: totalLiabilities,
		}

		c.JSON(http.StatusOK, jsonStruct)
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

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

	router.GET("/records", getRecords(db))

	router.POST("/record", saveRecord(db))

	router.DELETE("/record/:id", deleteRecord(db))

	router.Run(":" + port)
}
