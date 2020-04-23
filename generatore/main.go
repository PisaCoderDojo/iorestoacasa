package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

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
	interactive := flag.Bool("i", false, "Interactive Add Release")
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	prfBrowser := flag.String("b", "", "Preferred browser")
	flag.Parse()

	if *interactive {
		if err := add(*interactive); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := run(*prfBrowser, os.Stdout, *skipPreview); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func EncodeStringBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func readInput(rd *bufio.Reader, s string, pattern string, newCard []string, urlEconde bool) ([]string, error) {
	fmt.Print(s)
	text, err := rd.ReadString('\n')
	field := strings.TrimSpace(text)
	// either url or pattern
	if urlEconde {
		u, err := url.Parse(pattern)
		if err != nil {
			log.Fatal(err)
		}
		u.Path += field
		field = u.String()
	} else {
		field = fmt.Sprintf(pattern, field)
	}
	newCard = append(newCard, field)
	return newCard, err
}

func add(interactive bool) error {
	var newCard []string
	rd := bufio.NewReader(os.Stdin)
	fmt.Println("Add a new Card")
	newCard, err := readInput(rd, "Title: ", "%s", newCard, false)
	if err != nil {
		return err
	}
	newCard, err = readInput(rd, "Description: ", "%s", newCard, false)
	if err != nil {
		return err
	}
	newCard, err = readInput(rd, "YouTube ID: ", "https://www.youtube.com/embed/", newCard, true)
	if err != nil {
		return err
	}
	newCard, err = readInput(rd, "Slides filename: ", "https://github.com/PisaCoderDojo/dojo-slides/raw/master/iorestoacasa/", newCard, true)
	if err != nil {
		return err
	}
	// stars field is not yet used, here for PR compatibility
	newCard = append(newCard, "100")
	err = writeCSV(newCard)
	if err != nil {
		return err
	}
	return nil
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

	r.Comma = '|'

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

func writeCSV(record []string) error {
	csvfile, err := os.Open("data.csv") //, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	r := csv.NewReader(csvfile)

	r.Comma = '|'
	records, err := r.ReadAll()

	records = append([][]string{record}, records...)

	csvfile, err = os.OpenFile("data.csv", os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(csvfile)
	writer.Comma = '|'
	defer writer.Flush()
	writer.WriteAll(records)

	if err != nil {
		return err
	}

	return nil
}
