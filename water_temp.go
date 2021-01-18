// Reads temperature data from a DS18B20 into a RaspberryPi
// https://www.adafruit.com/product/642
//
// Info on what commands to run and what the data looks like:
// https://raspberrypi.stackexchange.com/questions/14978/dallas-1-wire-temperature-sensor-not-working
// TO get this running:
// sudo modprobe w1-gpio
// sudo modprobe w1-therm
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

// 1-wire stuff
// this file contains data that looks like this:
// Note: if YES is NO, wait 10s and try again.
/*
a2 00 4b 46 7f ff 0e 10 e5 : crc=e5 YES
a2 00 4b 46 7f ff 0e 10 e5 t=10125
*/
const probeDataDir = "/sys/bus/w1/devices/28-000006af39c9" // dir specifically for my temp probe
const probeDataFile = "w1_slave"                           // unfortunate. Linux's 1-wire lib names it this way.

// data comes in as an int of celcius * 1000
// first get celcius, then convert to farenheit
const probeMultiplier = 1000

func rawProbeDataToFarenheit(raw int) float64 {
	celcius := float64(raw) / probeMultiplier
	return cToF(celcius)
}

func cToF(c float64) float64 {
	return c*9/5 + 32.9
}

func main() {
	// open file
	path := probeDataDir + "/" + probeDataFile
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("could not open %v: %v\n", path, err)
	}
	// TODO do this not in defer if looping
	defer f.Close()

	// read contents into string
	data := make([]byte, 100)
	if _, err = f.Read(data); err != nil {
		log.Fatal("could not read file %v: %v\n", path, err)
	}

	// regexp see if YES or NO
	isValid := regexp.MustCompile(`YES|NO$`)
	good := isValid.Find(data)

	// if no, sleep 10s
	if string(good) == "NO" {
		log.Fatalf("data is not good: %q", data)
		// fmt.Println("no good. Sleeping 10s")
		// time.Sleep(10 * time.Second)
	}

	// if YES read `t=10125`
	if string(good) == "YES" {
		// match `t-10125` but capture the number after t= in a submatch group
		rawDataRe := regexp.MustCompile(`t=(\d{4,5})`)
		matches := rawDataRe.FindSubmatch(data)
		if len(matches) == 2 {
			// get submatch 1, which finds what's inside the regex parens
			rawData := string(matches[1])

			// convert from string to int
			rawNum, err := strconv.Atoi(rawData)
			if err != nil {
				log.Fatalf("rawData cannot be converted into number %q: %v", matches, err)
			}

			if err := storeReading(rawNum); err != nil {
				log.Fatalf("could not store rawNum %d: %v", rawNum, err)
			}

			// print degrees farenheit
			fmt.Println(rawProbeDataToFarenheit(rawNum))
		}
	}
}

// database file
const dbFile = "/home/pi/water.csv"

type reading struct {
	time  int64
	value int
}

// 1611012127,10125
func (r reading) String() string {
	return fmt.Sprintf("%d,%d\n", r.time, r.value)
}

func storeReading(value int) error {
	r := reading{
		time:  time.Now().Unix(),
		value: value,
	}

	// open and append to the file
	f, err := os.OpenFile(dbFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Could not open db file %v", err)
	}
	if _, err := f.Write([]byte(r.String())); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return fmt.Errorf("Could not write %q to db %v", r.String(), err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("Could not close db file %v", err)
	}

	return nil
}
