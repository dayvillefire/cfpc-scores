package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	debug     = flag.Bool("debug", false, "Debugging enabled")
	ssn       = flag.String("ssn", "", "Last 4 of SSN")
	date      = flag.String("date", "", "Date of test")
	dummydata = flag.String("dummy-data", "", "File with test data to parse")
)

func main() {
	flag.Parse()

	if *ssn == "" || *date == "" {
		flag.PrintDefaults()
		return
	}

	dt, err := time.Parse("01/02/2006", *date)
	if err != nil {
		panic(err)
	}

	body, w, err := dataToWriter(map[string]string{
		"vt_SSN":       *ssn,
		"vd_EntryDate": *date,                   // 05/21/2023
		"vd_QueryDate": dt.Format("2006-01-02"), // 2023-05-21
		"captcha_code": "a",
		"vt_LastName":  "",
		"vi_Search":    "Submit",
	})

	cl := http.DefaultClient
	cl.Timeout = time.Second * 5
	if *debug {
		log.Printf("DEBUG: Post")
	}
	req, err := http.NewRequest("POST", "http://www.cfpc-ct.info/cert-score-query.php", bytes.NewReader(body.Bytes()))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Origin", "http://www.cfpc-ct.info")
	req.Header.Set("Referer", "http://www.cfpc-ct.info/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Content-Type", w.FormDataContentType())

	if *dummydata != "" {
		fp, err := os.Open(*dummydata)
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		parse(fp)
		return
	}

	res, err := cl.Do(req)
	if err != nil {
		panic(err)
	}

	if *debug {
		b, _ := ioutil.ReadAll(res.Body)
		fmt.Printf("%s\n", string(b))
	}

	parse(res.Body)

}

func dataToWriter(in map[string]string) (*bytes.Buffer, *multipart.Writer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range in {
		fw, err := writer.CreateFormField(k)
		if err != nil {
			return body, writer, err
		}
		_, err = io.Copy(fw, strings.NewReader(v))
		if err != nil {
			return body, writer, err
		}
	}
	err := writer.Close()
	return body, writer, err
}

func parse(r io.Reader) {
	if *debug {
		log.Printf("DEBUG: parse()")
	}
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	if *debug {
		log.Printf("DEBUG: %#v", doc)
	}

	// Find the review items
	review := [][]string{}
	doc.Find("form#score_query table tr").Each(func(i int, s *goquery.Selection) {
		if *debug {
			log.Printf("DEBUG: %#v", s)
		}
		x := []string{}
		s.Find("td").Each(func(i2 int, s2 *goquery.Selection) {
			x = append(x, s2.Text())
			if *debug {
				log.Printf("DEBUG: %s", s2.Text())
			}
		})

		review = append(review, x)
	})

	for _, v := range review {
		if len(v) >= 4 {
			fmt.Printf("%s (%s) : %s\n", v[2], v[3], v[4])
		}
	}

}
