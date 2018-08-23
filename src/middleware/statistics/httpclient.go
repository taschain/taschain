package statistics

import (
	"fmt"
	"net/http"
	"time"
	"bytes"
	"log"
)

var url string

var timeout time.Duration

func SendPost(obj *bytes.Buffer) {

	timeout := time.Duration(1 * time.Second)
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