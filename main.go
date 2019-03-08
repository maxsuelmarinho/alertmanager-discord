package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/prometheus/alertmanager/template"
)

type alertManAlert struct {
	Annotations struct {
		Description string `json:"description"`
		Summary     string `json:"summary"`
	} `json:"annotations"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	StartsAt     string            `json:"startsAt"`
	Status       string            `json:"status"`
}

type alertManOut struct {
	Alerts            []alertManAlert `json:"alerts"`
	CommonAnnotations struct {
		Summary string `json:"summary"`
	} `json:"commonAnnotations"`
	CommonLabels struct {
		Alertname string `json:"alertname"`
	} `json:"commonLabels"`
	ExternalURL string `json:"externalURL"`
	GroupKey    string `json:"groupKey"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
	} `json:"groupLabels"`
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Version  string `json:"version"`
}

type discordAuthor struct {
	Name    string `json:"name"`
	IconUrl string `json:"icon_url"`
}

type discordFooter struct {
	Text string `json:"text"`
}

type discordField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Author      discordAuthor  `json:"author"`
	Description string         `json:"description"`
	Fields      []discordField `json:"fields"`
	Color       int            `json:"color"`
	Footer      discordFooter  `json:"footer"`
}

type discordRequest struct {
	Content   string         `json:"content"`
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url"`
	Embeds    []discordEmbed `json:"embeds"`
}

func main() {
	webhookURL := os.Getenv("DISCORD_WEBHOOK")
	whURL := flag.String("webhook.url", webhookURL, "")
	flag.Parse()

	if webhookURL == "" && *whURL == "" {
		log.Fatalf("error: environment variable DISCORD_WEBHOOK not found")
		os.Exit(1)
	}

	log.Printf("info: Listening on 0.0.0.0:9094")
	http.ListenAndServe(":9094", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		printRequest(r)

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		alertManagerNotification := template.Data{}
		err = json.Unmarshal(b, &alertManagerNotification)
		if err != nil {
			panic(err)
		}

		groupedAlerts := make(map[string][]template.Alert)

		for _, alert := range alertManagerNotification.Alerts {
			groupedAlerts[alert.Status] = append(groupedAlerts[alert.Status], alert)
		}

		for status, alerts := range groupedAlerts {
			request := discordRequest{
				Username:  "Prometheus",
				AvatarURL: "https://avatars1.githubusercontent.com/u/3380462?s=200&v=4",
				Embeds:    make([]discordEmbed, 0),
			}

			var iconUrl string
			var color int
			if strings.ToUpper(status) == "FIRING" {
				iconUrl = "https://www.iconfinder.com/icons/116853/download/png/128"
				color = 0
			} else {
				iconUrl = "https://www.iconfinder.com/icons/2682848/download/png/128"
				color = 255
			}

			for _, alert := range alerts {

				embed := discordEmbed{
					Author: discordAuthor{
						Name:    strings.ToUpper(status),
						IconUrl: iconUrl,
					},
					Title:       fmt.Sprintf("**%s**", alert.Annotations["summary"]),
					Description: alert.Annotations["description"],
					Color:       color,
					Fields:      make([]discordField, 0),
					Footer: discordFooter{
						Text: "",
					},
				}

				var value string
				for k, v := range alert.Labels {
					value += fmt.Sprintf("**%s:** %s\n", k, v)
				}

				field := discordField{
					Name:  "Labels:",
					Value: value,
				}

				embed.Fields = append(embed.Fields, field)

				request.Embeds = append(request.Embeds, embed)
			}

			requestJSON, _ := json.Marshal(request)
			response, err := http.Post(*whURL, "application/json", bytes.NewReader(requestJSON))
			if err != nil {
				log.Fatalf("failed to call discord webhook: %v", err)
			}
			var body string
			bodyBytes, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				body = fmt.Sprintf("failed read discord webhook response body: %v", err)
			} else {
				body = string(bodyBytes)
			}
			log.Printf("Discord webhook response: [httpStatus: %d; body: %s]", response.StatusCode, body)
		}
	}))
}

func printRequest(r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	log.Printf("Request received: %s", string(requestDump))
}
