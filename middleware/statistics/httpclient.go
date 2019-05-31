package statistics

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"
)

var url string

var timeout time.Duration

func SendPost(obj *bytes.Buffer, code string) {

	timeout := time.Duration(1 * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	request, err := http.NewRequest("POST", url, obj)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("type", code)
	_, err = client.Do(request)
	if err != nil {
		fmt.Println(err)
	}
}
