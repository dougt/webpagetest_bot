package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
		fmt.Printf("Not configured.  Could not find config.json\n")
		os.Exit(-1)
	}

	err = json.Unmarshal(data, &gServerConfig)
	if err != nil {
		fmt.Printf("Could not unmarshal config.json (%s)\n", err)
		os.Exit(-1)
		return
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
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// just search for the SpeedIndex with a regex.
	// Some day we can be smarter.

	testIdRegEx := regexp.MustCompile(`<testId>(.*)</testId>`)
	r := testIdRegEx.FindSubmatch(contents)
	testId := string(r[1])

	addTestIdToDb(conn, testId)
}

func updateSpeedIndex(conn *sqlite.Conn) {

	selectStmt, err := conn.Prepare("SELECT testId FROM testRuns WHERE speedIndex='-1';")
	if err != nil {
		fmt.Println("Error while creating selectSmt: %s", err)
		return
	}

	err = selectStmt.Exec()
	if err != nil {
		fmt.Println("Error while Selecting: %s", err)
		return
	}

	for selectStmt.Next() {
		fmt.Println("next... \n;")

		var testId = ""
		err = selectStmt.Scan(&testId)
		if err != nil {
			fmt.Printf("Error while getting row data: %s\n", err)
			return
		}
		fmt.Printf("Id => %s\n", testId)

		requestSpeedIndex(conn, testId)
	}
}

func requestSpeedIndex(conn *sqlite.Conn, testId string) {

	url := "http://www.webpagetest.org/xmlResult/" + testId + "/"

	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// just search for the SpeedIndex with a regex.
	// Some day we can be smarter.

	speedIndexRegEx := regexp.MustCompile(`<SpeedIndex>(.*)</SpeedIndex>`)
	r := speedIndexRegEx.FindSubmatch(contents)
	if r == nil {
		fmt.Printf("speed index not found\n")
		return
	}

	speedIndex := string(r[1])
	fmt.Printf("speed index:  %+v\n", speedIndex)

	if err != nil {
		fmt.Printf("error: %x", err)
		return
	}

	err = conn.Exec("UPDATE testRuns SET speedIndex=" + speedIndex + " WHERE testId='" + testId + "'")
	if err != nil {
		fmt.Println("Error while update: %s", err)
	}
}

func addTestIdToDb(conn *sqlite.Conn, testId string) {

	fmt.Printf("Adding test id: " + testId + " to db.\n")
	now := time.Now()
	err := conn.Exec("INSERT INTO testRuns(testId, date, speedIndex) VALUES('" + testId + "', '" + now.String() + "', '" + "-1" + "');")
	if err != nil {
		fmt.Println("Error while Inserting: %s", err)
	}
}

func dumpDatabase(conn *sqlite.Conn) {

	selectStmt, err := conn.Prepare("SELECT testId, date, speedIndex FROM testRuns;")
	if err != nil {
		fmt.Println("Error while creating selectSmt: %s", err)
		return
	}

	err = selectStmt.Exec()
	if err != nil {
		fmt.Println("Error while Selecting: %s", err)
		return
	}

	for selectStmt.Next() {
		var testId = ""
		var date = ""
		var speedIndex = ""

		err = selectStmt.Scan(&testId, &date, &speedIndex)
		if err != nil {
			fmt.Printf("Error while getting row data: %s\n", err)
			return
		}
		fmt.Printf("%s, %s, %s\n", testId, date, speedIndex)
	}
}

func main() {

	var getSpeedIndex = flag.Bool("get", false, "get speedIndex")
	var newTest = flag.Bool("create", false, "create test run")
	var dump = flag.Bool("dump", false, "dump dataset")

	flag.Parse();


	db := "tests.db"
	conn, err := sqlite.Open(db)
	if err != nil {
		fmt.Println("Unable to open the database: %s", err)
		os.Exit(1)
	}
	conn.Exec("CREATE TABLE testRuns(id INTEGER PRIMARY KEY AUTOINCREMENT, testId TEXT, date TEXT, speedIndex INT);")
	defer conn.Close()

	if *newTest {
		kickOffTest(conn);
	}

	if *getSpeedIndex {
		updateSpeedIndex(conn)
	}

	if *dump {
		dumpDatabase(conn)
	}
}
