package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	_ "github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Record struct {
	Name       string          `json:"name"`
	Amount     decimal.Decimal `json:"amount"`
	RecordType string          `json:"record_type"`
	Id         int             `json:"id"`
}

type myJSON struct {
	Records          []Record        `json:"records"`
	Totals           decimal.Decimal `json:"totals"`
	TotalLiabilities decimal.Decimal `json:"total_liabilities"`
	TotalAssets      decimal.Decimal `json:"total_assets"`
}

func saveRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
		}

		var newRecord Record
		err = json.Unmarshal(body, &newRecord)
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error saving record: %q", err))
			return
		}

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
			total            decimal.Decimal
			totalAssets      decimal.Decimal
			totalLiabilities decimal.Decimal
			records          []Record
		)

		rows, err := db.Query("SELECT * FROM records")
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error reading records db: %q", err))
			return
		}

		defer rows.Close()

		for rows.Next() {
			var (
				amount  decimal.Decimal
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

			if recType == "Asset" {
				totalAssets = totalAssets.Add(amount)
				total = total.Add(amount)
			}
			if recType == "Liability" {
				totalLiabilities = totalLiabilities.Add(amount)
				total = total.Sub(amount)
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

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS records (id serial PRIMARY KEY,name text, recType varchar(10), amount NUMERIC(20,6))"); err != nil {
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
