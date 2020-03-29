package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	texttemplate "text/template"
	"time"
)

type content struct {
	Body template.HTML
}

type card struct {
	Title   string
	Text    string
	YouTube string
	Slides  string
}

func main() {
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	prfBrowser := flag.String("b", "", "Preferred browser")
	flag.Parse()

	if err := run(*prfBrowser, os.Stdout, *skipPreview); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(prfBrowser string, out io.Writer, skipPreview bool) error {
	inputPage, err := ioutil.ReadFile("template-page.html.tmpl")
	if err != nil {
		return err
	}

	inputCard, err := ioutil.ReadFile("template-card.html.tmpl")
	if err != nil {
		return err
	}

	outName := "../index.html"

	htmlData, err := parseContent(inputPage, inputCard, outName)
	if err != nil {
		return err
	}

	if err := saveHTML(outName, htmlData); err != nil {
		return err
	}
	if skipPreview {
		return nil
	}

	return preview(outName, prfBrowser)
}

func parseContent(inputPage []byte, inputCard []byte, outName string) ([]byte, error) {

	t, err := template.New("dojo").Parse(string(inputPage))
	if err != nil {
		return nil, err
	}

	body, err := readCsv(inputCard)
	if err != nil {
		return nil, err
	}

	c := content{
		Body: template.HTML(body),
	}

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, c); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func saveHTML(outFname string, data []byte) error {
	return ioutil.WriteFile(outFname, data, 0644)
}

func preview(fname string, prfBrowser string) error {
	browser := "firefox"
	if prfBrowser != "" {
		browser = prfBrowser
	}

	browserPath, err := exec.LookPath(browser)
	if err != nil {
		return err
	}

	if err := exec.Command(browserPath, fname).Start(); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)
	return nil
}

func readCsv(inputCard []byte) ([]byte, error) {
	csvfile, err := os.Open("data.csv")
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(csvfile)

	r.Comma = ';'

	t, err := texttemplate.New("card").Parse(string(inputCard))
	if err != nil {
		return nil, err
	}
	var cards []byte
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Title: %s Text: %s Youtube: %s Slides: %s\n", record[0], record[1], record[2], record[3])
		c := card{
			Title:   record[0],
			Text:    record[1],
			YouTube: record[2],
			Slides:  record[3],
		}

		var buffer bytes.Buffer
		if err := t.Execute(&buffer, c); err != nil {
			return nil, err
		}
		cards = append(cards, buffer.Bytes()...)
	}
	return cards, nil
}
