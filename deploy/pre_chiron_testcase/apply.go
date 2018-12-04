package main

import (
	"os"
	"bufio"
	"strings"
	"io"
	"bytes"
	"time"
	"net/http"
	"log"
	"fmt"
)

func SendPost(obj *bytes.Buffer, url string) {
	timeout := time.Duration(2 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	request, err := http.NewRequest("POST", url, obj)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = client.Do(request)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	f, err := os.Open("host_ids")
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)
	ips := make([]string,0)
	for {
		line, err := buf.ReadString('\n')

		array := strings.Split(line," ")
		ips = append(ips,array[0])
		if err != nil {
			if err == io.EOF {
				break
			} else{
				panic(err)
			}
		}
	}
}