package util

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-contrib/sse"
)

var NotFound error = errors.New("404 Not Found")

func fetchData(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, NotFound
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v", resp.Status)
	}

	return resp, nil
}
func FetchDataToBytes(url string) ([]byte, error) {
	empty := []byte("")
	resp, err := fetchData(url)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}

	return content, nil
}
func FetchDataSSE(url string, done chan struct{}, cb func(sse.Event), on_done func()) error {
	resp, err := fetchData(url)
	if err != nil {
		return err
	}

	go func() {
		buf := BufPool.Get()
		defer BufPool.Put(buf)
		defer on_done()
		defer resp.Body.Close()
		b := make([]byte, 64)
		for {
			n, err := resp.Body.Read(b)
			if err != nil {
				if err == io.EOF {
					break
				}
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				fmt.Printf("Could not read data: %v\n", err)
				break
			}
			data := b[:n]
			consecutive_newlines := 0
			for i, c := range data {
				if c == '\n' {
					consecutive_newlines += 1
				} else if c != '\r' {
					consecutive_newlines = 0
				}

				if consecutive_newlines == 2 {
					_, err = buf.Write(data[:i+1])
					if err != nil {
						fmt.Printf("%v\n", err)
						return
					}

					events, err := sse.Decode(buf)
					if err != nil {
						fmt.Printf("%v\n", err)
						return
					}

					for _, event := range events {
						cb(event)
					}

					buf.Reset()

					data = data[i+1:]

					break
				}
			}

			_, err = buf.Write(data)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
		}
	}()

	go func() {
		<-done
		resp.Body.Close()
	}()

	return nil
}
