package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type ServerConfig struct {
	key string `json:"key"`
}

var gServerConfig ServerConfig

func readConfig() {

	var data []byte
	var err error

	data, err = ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Not configured.  Could not find config.json")
	}

	err = json.Unmarshal(data, &gServerConfig)
	if err != nil {
		log.Fatalf("Could not unmarshal config.json: ", err)
	}
}

func kickOffTest(conn *sqlite.Conn) {

	location := "Dulles:Firefox.DSL"
	testurl := url.QueryEscape("http://www.google.com/search?hl=en&q=mozilla+foundation")
	runs := string(2)
	key := gServerConfig.key
	url := "https://www.webpagetest.org/runtest.php?url=" + testurl + "&runs=" + runs + "&f=xml&k=" + key + "&location=" + location

	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("%v", err)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// just search for the SpeedIndex with a regex.
	// Some day we can be smarter.

	testIdRegEx := regexp.MustCompile(`<testId>(.*)</testId>`)
	r := testIdRegEx.FindSubmatch(contents)
	testId := string(r[1])

	addTestIdToDb(conn, testId)
}

func updateSpeedIndex(conn *sqlite.Conn) {

	// Find all test runs that do not have a speedIndex yet (they are set to -1)
	selectStmt, err := conn.Prepare("SELECT testId FROM testRuns WHERE speedIndex='-1';")
	if err != nil {
		log.Fatalf("Error while preparing select: ", err)
	}

	err = selectStmt.Exec()
	if err != nil {
		log.Fatalf("Error while exec select: ", err)
	}

	for selectStmt.Next() {

		var testId = ""
		err = selectStmt.Scan(&testId)
		if err != nil {
			log.Fatalf("Error while getting row data: ", err)
		}

		requestSpeedIndex(conn, testId)
	}
}

func requestSpeedIndex(conn *sqlite.Conn, testId string) {

	url := "http://www.webpagetest.org/xmlResult/" + testId + "/"

	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("%v", err)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// just search for the SpeedIndex with a regex.
	// Some day we can be smarter.

	speedIndexRegEx := regexp.MustCompile(`<SpeedIndex>(.*)</SpeedIndex>`)
	r := speedIndexRegEx.FindSubmatch(contents)
	if r == nil {
		log.Println("speed index not found for testId: ", testId)
		return
	}

	speedIndex := string(r[1])
	log.Println("speed index found for testId: ", testId, " speedIndex: ", speedIndex)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = conn.Exec("UPDATE testRuns SET speedIndex=" + speedIndex + " WHERE testId='" + testId + "'")
	if err != nil {
		log.Println("Error while update testId: "+testId+" err: ", err)
	}
}

func addTestIdToDb(conn *sqlite.Conn, testId string) {

	log.Println("Adding test id: " + testId + " to db.")
	now := time.Now()
	err := conn.Exec("INSERT INTO testRuns(testId, date, speedIndex) VALUES('" + testId + "', '" + now.String() + "', '" + "-1" + "');")
	if err != nil {
		log.Fatalf("Error while Inserting: ", err)
	}
}

func dumpDatabase(conn *sqlite.Conn) {

	selectStmt, err := conn.Prepare("SELECT testId, date, speedIndex FROM testRuns;")
	if err != nil {
		log.Fatalf("Error while creating selectSmt: ", err)
	}

	err = selectStmt.Exec()
	if err != nil {
		log.Fatalf("Error while Selecting: ", err)
	}

	for selectStmt.Next() {
		var testId = ""
		var date = ""
		var speedIndex = ""

		err = selectStmt.Scan(&testId, &date, &speedIndex)
		if err != nil {
			log.Fatalf("Error while getting row data: ", err)
		}

		// this should go somewhere else
		log.Println("%v, %v, %v", testId, date, speedIndex)
	}
}

func main() {

	var getSpeedIndex = flag.Bool("get", false, "get speedIndex")
	var newTest = flag.Bool("create", false, "create test run")
	var dump = flag.Bool("dump", false, "dump dataset")
	var debug = flag.Bool("debug", false, "enable debug logging")
	var logFile = flag.String("logFile", "", "path of log file")
	var databaseFile = flag.String("databaseFile", "tests.db", "path of sqlite database")

	flag.Parse()

	if *debug == false {
		log.SetOutput(ioutil.Discard)
	} else if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(f)
		}
	}

	db := *databaseFile
	conn, err := sqlite.Open(db)
	if err != nil {
		log.Fatalf("Unable to open the database: ", err)
	}
	conn.Exec("CREATE TABLE testRuns(id INTEGER PRIMARY KEY AUTOINCREMENT, testId TEXT, date TEXT, speedIndex INT);")
	defer conn.Close()

	if *newTest {
		kickOffTest(conn)
	}

	if *getSpeedIndex {
		updateSpeedIndex(conn)
	}

	if *dump {
		dumpDatabase(conn)
	}
}
