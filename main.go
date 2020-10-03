package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type cmdArgs struct {
	collections      []string
	out              string
	data             string
	ton              string
	threshold        float64
	filterStrictMode bool
}

func parseArgs() *cmdArgs {
	var collectionsStr string
	args := new(cmdArgs)
	flag.StringVar(&collectionsStr, "c", "", "collections concat-ed by comma. shijing,tangshi,songci are supported")
	flag.StringVar(&args.out, "o", "output", "output file")
	flag.StringVar(&args.data, "d", "", "data directory")
	flag.StringVar(&args.ton, "t", "", "ton, e.g. liu2")
	flag.Float64Var(&args.threshold, "b", 0.0, "influnce baseline [0, 1]")
	flag.BoolVar(&args.filterStrictMode, "s", false, "filter strict mode. all influnce has to be beyond the baseline")

	flag.Parse()

	args.collections = strings.Split(collectionsStr, ",")
	return args
}

func main() {
	args := parseArgs()
	log.Printf("args: %+v\n", args)
	fh, err := os.OpenFile(args.out, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println("open file error", args, err)
		panic(err)
	}
	defer fh.Close()

	out := bufio.NewWriter(fh)
	defer out.Flush()

	if args.ton == "" {
		log.Println("invlaid ton")
		panic(errors.New("invalid ton"))
	}
	if len(args.collections) == 0 {
		log.Println("empty collection list")
		panic(errors.New("empty collection list"))
	}
	if args.threshold > 1 || args.threshold < 0 {
		log.Println("bad baseline, must be [0, 1]")
		panic(errors.New("bad baseline"))
	}

	collections := []iCollection{}
	for _, colName := range args.collections {
		var (
			col iCollection
			err error
		)
		switch colName {
		case "shijing":
			col, err = newShijingCollection(args.data)
		case "tang":
			col, err = newTangCollection(args.data)
		case "song":
			col, err = newSongCollection(args.data)
		default:
			err = fmt.Errorf("unknown collection %s", col)
		}

		if err != nil {
			log.Println("open collection error", err)
			panic(err)
		}

		log.Println("adding", col.name())
		collections = append(collections, col)
	}

	for _, col := range collections {
		log.Println("processing collection", col.name())

		pl := col.poetryList()
		for poetry := range pl {
			if poetry.filtered(args.threshold, args.filterStrictMode) {
				continue
			}

			lines, hit := poetry.containsTonLines(args.ton)
			if !hit {
				continue
			}

			writelines(out, poetry.headline(), "")
			writelines(out, lines...)
			writelines(out, "\n", "\n")
		}

		out.Flush()
	}

}

func writelines(out *bufio.Writer, lines ...string) {
	if len(lines) == 0 {
		return
	}

	lines = append(lines, "")
	out.WriteString(strings.Join(lines, "\n"))
}
