package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hoisie/web"
)

const (
	port = "0.0.0.0:7007"
)

type Vnstat struct {
	Ifaces Iface `xml:"interface"`
}

type Iface struct {
	Id      string      `xml:"id"`
	Nick    string      `xml:"nick"`
	Created DateTime    `xml:"created"`
	Updated DateTime    `xml:"updated"`
	Traffic TrafficType `xml:"traffic"`
}

type DateTime struct {
	DateVal Date `xml:"date"`
	TimeVal Time `xml:"time"`
}

type DateTimeData struct {
	DateVal  Date   `xml:"date"`
	TimeVal  Time   `xml:"time"`
	Transfer string `xml:"tx"`
	Receive  string `xml:"rx"`
}

type TrafficType struct {
	Total  TotalType  `xml:"total"`
	Days   DaysType   `xml:"days"`
	Months MonthsType `xml:"months"`
	Tops   TopsType   `xml:"tops"`
	Hours  HoursType  `xml:"hours"`
}

type DaysType struct {
	Day []DateTimeData `xml:"day"`
}

type MonthsType struct {
	Month []DateTimeData `xml:"month"`
}

type TopsType struct {
	Top []DateTimeData `xml:"top"`
}

type HoursType struct {
	Hour []DateTimeData `xml:"hour"`
}

type TotalType struct {
	Transfer string `xml:"tx"`
	Receive  string `xml:"rx"`
}

type Date struct {
	Year  string `xml:"year"`
	Month string `xml:"month"`
	Day   string `xml:"day"`
}

type Time struct {
	Hour   string `xml:"hour"`
	Minute string `xml:"minute"`
}

func runcmd(arg ...string) string {

	cmd := exec.Command(arg[0], arg[1], arg[2], arg[3])
	out, err := cmd.Output()

	if err != nil {
		fmt.Println("Error: %s", err.Error())
		return "" //TODO: Exception
	}

	return string(out)
}

func vnstat(ctx *web.Context, iface string, debug string) string {
	app := "vnstat"

	arg0 := "-i"
	arg2 := "--xml"

	ctx.SetHeader("Content-Type", "application/json", true)

	rawxml := runcmd(app, arg0, iface, arg2)

	var q Vnstat
	xml.Unmarshal([]byte(rawxml), &q)

	var jsondata []byte
	var err error

	if debug == "debug" {
		jsondata, err = json.MarshalIndent(q, "", "  ")
	} else {
		jsondata, err = json.Marshal(q)
	}

	if err != nil {
		return "Error"
	}
	return string(jsondata)
}

func dashboard(ctx *web.Context, iface string) string {
	app := "vnstat"

	arg0 := "-i"

	rawstat := runcmd(app, arg0, iface, "-ru")

	return rawstat
}

func ifacelist(ctx *web.Context) string {
	app := "sh"
	arg0 := "-c"
	arg1 := "ls `vnstat --showconfig | grep DatabaseDir | sed -E 's/.* \"(.*)\"/\\1/'`"

	ifaces := strings.TrimSpace(runcmd(app, arg0, arg1, ""))
	ifacelist := strings.Split(ifaces, "\n")

	jsondata, err := json.Marshal(ifacelist)

	if err != nil {
		return "Error"
	}

	return string(jsondata)
}

func home(ctx *web.Context) {
	ctx.Redirect(301, "/stat.html")
}

func monitorUsage(iface string, nlimit int, scmd string) {
	getTotalTransfer := func() (int, error) {
		app := "vnstat"

		arg0 := "-i"
		arg2 := "--xml"

		rawxml := runcmd(app, arg0, iface, arg2)

		var q Vnstat
		xml.Unmarshal([]byte(rawxml), &q)
		if len(q.Ifaces.Traffic.Months.Month) > 0 {
			return strconv.Atoi(q.Ifaces.Traffic.Months.Month[0].Transfer)
		} else {
			return 0, nil
		}
	}

	cmdSlice := strings.Fields(scmd)
	cmdExe := cmdSlice[0]
	cmdSlice = cmdSlice[1:]

	for {
		if n, e := getTotalTransfer(); e == nil && n >= nlimit {
			log.Println("Bandwidth limit exceed")
			out := exec.Command(cmdExe, cmdSlice...)
			ret, _ := out.Output()
			log.Printf("Result: %s\n", ret)
			return
		} else {
			log.Println("Current transfer:", float32(n)/1024/1024, "GiB")
		}
		time.Sleep(time.Minute * 5)
	}
}

func main() {
	var (
		iFace  string = ""
		nLimit int    = 0
		sCmd   string = ""
	)

	flag.StringVar(&iFace, "i", "", "monitored interface")
	flag.IntVar(&nLimit, "l", 0x8FFFFFF, "Bandwidth usage limit (in GiB)")
	flag.StringVar(&sCmd, "c", "shutdown", "Command to execute on exceed of bandwidth")
	flag.Parse()

	if iFace != "" {
		// bandwidth monitoring enabled
		log.Println("Starting monitor for", iFace, "with a limit of", nLimit, "GiB")
		go monitorUsage(iFace, nLimit*1024*1024, sCmd)
	}
	web.Get("/vnstat/(.*)/(.*)", vnstat)
	web.Get("/dashboard/(.*)", dashboard)
	web.Get("/list", ifacelist)
	web.Get("/", home)
	web.Run(port)
}
