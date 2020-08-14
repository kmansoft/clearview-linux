package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
)

type Flags struct {
	Template      string
	AgentVersion  string
	ServerVersion string
	Html          string
}

type Data struct {
	AgentVersion  string
	ServerVersion string
}

func main() {
	flags := Flags{}

	flag.StringVar(&flags.Template, "template", "", "Source template")
	flag.StringVar(&flags.AgentVersion, "agent-version", "", "Agent version")
	flag.StringVar(&flags.ServerVersion, "server-version", "", "Server version")
	flag.StringVar(&flags.Html, "html", "", "Output html file")

	flag.Parse()

	if flags.Template == "" ||
		flags.AgentVersion == "" ||
		flags.ServerVersion == "" ||
		flags.Html == "" {
		fmt.Printf("Please specify template, agent version, server version, html parameters\n")
		os.Exit(1)
	}

	fmt.Printf("Processing template: %s\n", flags.Template)
	fmt.Printf("Agent version name: %s\n", flags.AgentVersion)
	fmt.Printf("Server version name: %s\n", flags.ServerVersion)
	fmt.Printf("Output html: %s\n", flags.Html)

	templateData, err := ioutil.ReadFile(flags.Template)
	if err != nil {
		fmt.Printf("Error reading template %s: %v\n", flags.Template, err)
		os.Exit(1)
	}

	t, err := template.New("download").Parse(string(templateData))
	if err != nil {
		fmt.Printf("Error parsing template %s: %v\n", flags.Template, err)
		os.Exit(1)
	}

	buf := bytes.Buffer{}
	data := Data{
		AgentVersion:  flags.AgentVersion,
		ServerVersion: flags.ServerVersion,
	}
	err = t.Execute(&buf, &data)
	if err != nil {
		fmt.Printf("Error executing template %s: %v\n", flags.Template, err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(flags.Html, buf.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Error writing template to %s: %v\n", flags.Html, err)
		os.Exit(1)
	}

	fmt.Printf("Processed the template with no errors\n")
}
