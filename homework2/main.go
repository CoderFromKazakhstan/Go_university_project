package main

import (
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
)

type UserInput struct {
	from    int
	to      int
	workers int
	timeout int
}

type Result struct {
	Id       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
	Err      error  `json:"-"`
}

const defaultURL = "http://127.0.0.1"

func getDiscription(ctx context.Context, client *http.Client, tasks <-chan int, results chan<- Result, uIN UserInput, wg *sync.WaitGroup) {
	defer wg.Done()

	for id := range tasks {
		processTask(ctx, client, uIN, results, id)
	}
}

func processTask(ctx context.Context, client *http.Client, uIN UserInput, results chan<- Result, id int) {
	taskCTX, cancel := context.WithTimeout(ctx, time.Duration(uIN.timeout)*time.Second)
	defer cancel()

	url := fmt.Sprintf("%v/%v/info.0.json", getURL(), id)

	req, err := http.NewRequestWithContext(taskCTX, "GET", url, nil)
	if err != nil {
		results <- Result{Id: id, Err: err}
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		results <- Result{Id: id, Err: err}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			results <- Result{Id: id, Err: fmt.Errorf("cannot read statusCode %v", errRead)}
			return
		}
		errorMessage := string(respBody)
		errorMessage = strings.TrimSuffix(errorMessage, "\n")
		results <- Result{Id: id, Err: fmt.Errorf("%v, %v", resp.StatusCode, errorMessage)}
		return
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		results <- Result{Id: id, Err: err}
		return
	}
	result.Id = id

	results <- result

}

func inputErorrsHanler(uIN UserInput) {
	if uIN.from == -1 || uIN.to == -1 {
		flag.Usage()
		log.Fatal("not enaugh arguments")
	}

	if uIN.from > uIN.to {
		log.Fatal("invalid range")
	}

	if uIN.workers < 1 {
		log.Fatal("invalid amount of workers")
	}
}

func getURL() string {
	url := os.Getenv("MOVIE_BASE_URL")
	if url == "" {
		url = defaultURL
	}
	return url
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var uIN UserInput

	flag.IntVar(&uIN.from, "from", -1, "Required flag")
	flag.IntVar(&uIN.to, "to", -1, "Required flag")
	flag.IntVar(&uIN.workers, "workers", 10, "Optional flag")
	flag.IntVar(&uIN.timeout, "timeout", 5, "Optional flag")
	flag.Parse()

	inputErorrsHanler(uIN)

	fmt.Println("Started")

	tasks := make(chan int, uIN.workers)
	results := make(chan Result, uIN.workers)

	client := http.Client{}

	var wg sync.WaitGroup
	for w := 0; w < uIN.workers; w++ {
		wg.Add(1)
		go getDiscription(ctx, &client, tasks, results, uIN, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		defer close(tasks)
		for t := uIN.from; t <= uIN.to; t++ {
			select {
			case <-ctx.Done():
				return
			case tasks <- t:
			}
		}
	}()

	allResults := make([]Result, 0, uIN.to-uIN.from+1)
	for res := range results {
		allResults = append(allResults, res)
	}

	slices.SortFunc(allResults, func(a, b Result) int {
		return cmp.Compare(a.Id, b.Id)
	})

	for _, res := range allResults {
		if res.Err != nil {
			fmt.Printf("%v — %v\n", res.Id, res.Err)
		} else {
			fmt.Printf("%v — %v — %v — %v\n", res.Id, res.Title, res.Year, res.Director)
		}
	}
}
