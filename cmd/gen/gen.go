package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	log = flag.String("log", "../cli/benchmarks/test.log", "log file")
)

func main() {
	flag.Parse()

	f, err := os.Open(*log)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var fsyncTps = make(map[string]string, 11)
	var fsyncTime = make(map[string]string, 11)
	var nofsyncTps = make(map[string]string, 11)
	var nofsyncTime = make(map[string]string, 11)
	var fsyncnames = "| |"
	var fsyncsp = "|--|"
	var nofsyncnames = "| |"
	var nofsyncsp = "|--|"

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if strings.Contains(line, "main.go") {
			continue
		}

		// bbolt/nofsync set rate: 12005 op/s, mean: 2082 ns
		items := strings.Split(line, " ")
		name := items[0]
		i := strings.LastIndex(name, "/")
		if i < 0 {
			continue
		}
		fsync := name[i+1:]
		name = name[:i]
		op := items[1]
		tps := items[3]
		took := items[6]

		if fsync == "fsync" {
			v1 := fsyncTps[op]
			if v1 == "" {
				v1 = "|" + op + "|" + tps + "|"
			} else {
				v1 = v1 + tps + "|"
			}
			fsyncTps[op] = v1

			if op == "set" {
				fsyncnames = fsyncnames + name + "|"
				fsyncsp = fsyncsp + "--|"
			}

			v2 := fsyncTime[op]
			if v2 == "" {
				v2 = "|" + op + "|" + took + "|"
			} else {
				v2 = v2 + took + "|"
			}
			fsyncTime[op] = v2
		} else {
			v1 := nofsyncTps[op]
			if v1 == "" {
				v1 = "|" + op + "|" + tps + "|"
			} else {
				v1 = v1 + tps + "|"
			}
			nofsyncTps[op] = v1

			if op == "set" {
				nofsyncnames = nofsyncnames + name + "|"
				nofsyncsp = nofsyncsp + "--|"
			}

			v2 := nofsyncTime[op]
			if v2 == "" {
				v2 = "|" + op + "|" + took + "|"
			} else {
				v2 = v2 + took + "|"
			}
			nofsyncTime[op] = v2
		}
	}

	// print nosync throughputs
	fmt.Println("nofsync - throughputs\n")
	fmt.Println(nofsyncnames)
	fmt.Println(nofsyncsp)
	for _, v := range nofsyncTps {
		fmt.Println(v)
	}
	fmt.Println("\n")

	// print nosync time
	fmt.Println("nofsync - time\n")
	fmt.Println(nofsyncnames)
	fmt.Println(nofsyncsp)
	for _, v := range nofsyncTime {
		fmt.Println(v)
	}
	fmt.Println("\n")

	// print sync throughputs
	fmt.Println("fsync - throughputs\n")
	fmt.Println(fsyncnames)
	fmt.Println(fsyncsp)
	for _, v := range fsyncTps {
		fmt.Println(v)
	}
	fmt.Println("\n")

	// print sync time
	fmt.Println("fsync - time\n")
	fmt.Println(fsyncnames)
	fmt.Println(fsyncsp)
	for _, v := range fsyncTime {
		fmt.Println(v)
	}
}
