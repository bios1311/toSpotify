package client

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/publicsuffix"
)

// create a rest api using gin
func CreateClient() *http.Client {
	jar := createCookieJar()
	transPort := createTrasport()
	client := &http.Client{
		Timeout:   time.Duration(30 * time.Second),
		Jar:       jar,
		Transport: transPort,
	}
	log.Println("HTTP client created")
	return client
}

func createCookieJar() http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Printf("failed to create cookie jar (%v); continuing without one", err)
		return nil
	}
	return jar
}
func createTrasport() *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return transport
}
