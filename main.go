package main

import (
	"log"
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/streadway/amqp"
)

var totalTime int64 = 0
var totalCount int64 = 0

type MqMessage struct {
	TimeNow        time.Time
	SequenceNumber int
	Payload        string
}

func main() {
	app := cli.NewApp()
	app.Name = "tester"
	app.Usage = "Make the rabbit cry"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server, s", Value: "rabbit-mq-test.cs1cloud.internal", Usage: "Hostname for RabbitMQ server"},
		cli.StringFlag{Name: "port, P", Value: "5672", Usage: "Port for RabbitMQ server"},
		cli.StringFlag{Name: "user, u", Value: "guest", Usage: "user for RabbitMQ server"},
		cli.StringFlag{Name: "password, pass", Value: "guest", Usage: "user pasword for RabbitMQ server"},
		cli.StringFlag{Name: "queue, Q", Value: "stress-test-exchange", Usage: "Name of the queue to work with"},
		cli.IntFlag{Name: "producer, p", Value: 0, Usage: "Number of messages to produce, -1 to produce forever"},
		cli.IntFlag{Name: "wait, w", Value: 0, Usage: "Number of nanoseconds to wait between publish events"},
		cli.IntFlag{Name: "consumer, c", Value: -1, Usage: "Number of messages to consume. 0 consumes forever"},
		cli.IntFlag{Name: "bytes, b", Value: 0, Usage: "number of extra bytes to add to the RabbitMQ message payload. About 50K max"},
		cli.IntFlag{Name: "concurrency, n", Value: 50, Usage: "number of reader/writer Goroutines"},
		cli.BoolFlag{Name: "quiet, q", Usage: "Print only errors to stdout"},
		cli.BoolFlag{Name: "wait-for-ack, a", Usage: "Wait for an ack or nack after enqueueing a message"},
	}
	app.Action = func(c *cli.Context) error {
		runApp(c)
		return nil
	}
	app.Run(os.Args)
}

func runApp(c *cli.Context) {
	println("Running!")
	porto := "amqp://"
	uri := porto + c.String("user") + ":" + c.String("password") + "@" + c.String("server") + ":" + c.String("port")

	if c.Int("consumer") > -1 {
		makeConsumers(uri, c.String("queue"), c.Int("concurrency"), c.Int("consumer"))
	}

	if c.Int("producer") != 0 {
		config := ProducerConfig{uri, c.String("queue"), c.Int("bytes"), c.Bool("quiet"), c.Bool("wait-for-ack")}
		makeProducers(c.Int("producer"), c.Int("wait"), c.Int("concurrency"), config)
	}
}

func MakeQueue(c *amqp.Channel, queueName string) amqp.Queue {
	q, err := c.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	return q
}

func makeProducers(n int, wait int, concurrency int, config ProducerConfig) {

	taskChan := make(chan int)
	for i := 0; i < concurrency; i++ {
		go Produce(config, taskChan)
	}

	start := time.Now()

	for i := 0; i < n; i++ {
		taskChan <- i
		time.Sleep(time.Duration(int64(wait)))
	}

	time.Sleep(time.Duration(10000))

	close(taskChan)

	log.Printf("Finished: %s", time.Since(start))
}

func makeConsumers(uri string, queue string, concurrency int, toConsume int) {

	doneChan := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go Consume(uri, queue, doneChan)
	}

	start := time.Now()

	if toConsume > 0 {
		for i := 0; i < toConsume; i++ {
			<-doneChan
			if i == 1 {
				start = time.Now()
			}
			log.Println("Consumed: ", i)
		}
	} else {

		for {
			<-doneChan
		}
	}

	log.Printf("Done consuming! %s", time.Since(start))
}
