package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	backend "CurrencyAlertApi/Backend"
	m "CurrencyAlertApi/Model"

	"github.com/gofiber/fiber/v2"
	"github.com/streadway/amqp"

	jwtware "github.com/gofiber/jwt/v3"
)

//Routers function used to route each request to appropriate url
func routers(app *fiber.App) {
	app.Post("/login", backend.Login)

	app.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte("secret"),
	}))

	app.Get("/alerts/create/:curr/:target", backend.AlertCreate)
	app.Get("/alerts/delete/:id", backend.AlertDelete)
	app.Get("/fetchall", backend.FetchAlerts)
	app.Get("/paginatedfetch", backend.FetchPaginatedAlerts)

}

func ticker() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Minute)
	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=USD&order=market_cap_desc&per_page=100&page=1&sparkline=false"
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"task_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue")
	for range ticker.C {

		response, err := http.Get(url)
		if err != nil {
			fmt.Print(err.Error())
		}

		responseData, err := ioutil.ReadAll(response.Body)

		if err != nil {
			log.Fatal(err)
		}
		var responseObject []m.Currencies
		json.Unmarshal(responseData, &responseObject)

		for _, v := range responseObject {
			backend.UpdateCurrencies(v.Symbol, v.Name, v.CurrentPrice)
		}

		for _, v := range backend.GetAllOngoingAlerts() {
			currency := backend.GetCurrency(v.Currency)
			if len(currency.Symbol) != 0 && v.Target == currency.CurrentPrice && v.Triggered == "false" {
				backend.TriggerAlert(v.ID)
				body := fmt.Sprintf("%s,%s,%g", v.Email, currency.Symbol, v.Target)
				err = ch.Publish(
					"",     // exchange
					q.Name, // routing key
					false,  // mandatory
					false,
					amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						ContentType:  "text/plain",
						Body:         []byte(body),
					})
				failOnError(err, "Failed to publish a message")
				log.Printf(" [x] Sent %s", body)

			}
		}

	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {

	backend.InitialMigration()
	app := fiber.New()
	go ticker()
	routers(app)

	app.Listen(":3000")
}
