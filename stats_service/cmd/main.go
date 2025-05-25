package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	protostats "statsservice/proto/stats"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

type Stat struct {
	EventType string `json:"event_type"`
	UserID    string `json:"user_id"`
	ObjectID  string `json:"object_id"`
	PromoId   string `json:"promo_id"`
	Timestamp int64  `json:"timestamp"`
}

func StartConsumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"kafka:9092"},
		Topic:     "stats",
		GroupID:   "stats-consumer-group",
		Partition: 0,
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	log.Println("Stats service started. Listening for messages...")

	go func() {
		for {
			msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Could not read message: %v", err)
				continue
			}
			log.Printf("Received: %s", string(msg.Value))
			var s Stat
			if err := json.Unmarshal(msg.Value, &s); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}
			_, err = DB.Exec(
				`INSERT INTO stats (event_type, user_id, object_id, promo_id, timestamp) VALUES (?, ?, ?, ?, ?)`,
				s.EventType, s.UserID, s.ObjectID, s.PromoId, time.Unix(s.Timestamp, 0),
			)
			if err != nil {
				log.Println("ClickHouse insert error:", err)
			}
		}
	}()
}

var DB *sql.DB

func InitClickhouse() error {
	var err error
	dsn := "tcp://clickhouse:9000?debug=true"

	for i := 0; i < 10; i++ {
		DB, err = sql.Open("clickhouse", dsn)
		if err != nil {
			log.Printf("ClickHouse open error (attempt %d): %v", i+1, err)
		} else if pingErr := DB.Ping(); pingErr != nil {
			log.Printf("ClickHouse ping error (attempt %d): %v", i+1, pingErr)
			err = pingErr
		} else {
			log.Println("✅ ClickHouse connected")

			createTableQuery := `
				CREATE TABLE IF NOT EXISTS stats (
					object_id String,
					promo_id String,
					user_id String,
					event_type String,
					timestamp DateTime
				)
				ENGINE = MergeTree
				PARTITION BY toYYYYMM(timestamp)
				ORDER BY (object_id, event_type, timestamp)
			`
			if _, err := DB.Exec(createTableQuery); err != nil {
				return fmt.Errorf("failed to create stats table: %w", err)
			}

			return nil
		}

		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("failed to connect to ClickHouse after retries: %w", err)
}

type Server struct {
	protostats.UnimplementedStatsServiceServer
}

func (s *Server) GetPromoStats(ctx context.Context, req *protostats.PromoRequest) (*protostats.PromoStats, error) {
	var views, clicks, comments int32
	query := `
		SELECT
			countIf(event_type = 'promo_viewed') AS views,
			countIf(event_type = 'promo_click') AS clicks,
			countIf(event_type = 'comment_published') AS comments
		FROM stats
		WHERE promo_id = ?
	`
	err := DB.QueryRow(query, req.PromoId).Scan(&views, &clicks, &comments)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &protostats.PromoStats{
		Views:    views,
		Clicks:   clicks,
		Comments: comments,
	}, nil
}

func getDailyStats(promoID, eventType string) ([]*protostats.DailyData, error) {
	query := `
		SELECT
			toDate(timestamp) AS day,
			count() AS cnt
		FROM stats
		WHERE promo_id = ? AND event_type = ?
		GROUP BY day
		ORDER BY day
	`
	rows, err := DB.Query(query, promoID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*protostats.DailyData
	for rows.Next() {
		var date time.Time
		var count int32
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		stats = append(stats, &protostats.DailyData{
			Date:  date.Format("2006-01-02"),
			Count: count,
		})
	}
	return stats, nil
}

func (s *Server) GetPromoViews(ctx context.Context, req *protostats.PromoRequest) (*protostats.DailyStats, error) {
	stats, err := getDailyStats(req.PromoId, "promo_viewed")
	if err != nil {
		return nil, err
	}
	return &protostats.DailyStats{Stats: stats}, nil
}

func (s *Server) GetPromoClicks(ctx context.Context, req *protostats.PromoRequest) (*protostats.DailyStats, error) {
	stats, err := getDailyStats(req.PromoId, "promo_click")
	if err != nil {
		return nil, err
	}
	return &protostats.DailyStats{Stats: stats}, nil
}

func (s *Server) GetPromoComments(ctx context.Context, req *protostats.PromoRequest) (*protostats.DailyStats, error) {
	stats, err := getDailyStats(req.PromoId, "comment_published")
	if err != nil {
		return nil, err
	}
	return &protostats.DailyStats{Stats: stats}, nil
}

func (s *Server) GetTopPromos(ctx context.Context, req *protostats.TopRequest) (*protostats.TopStats, error) {
	eventType, ok := eventTypeMap(req.Metric)
	if !ok {
		return nil, fmt.Errorf("invalid metric: %s", req.Metric)
	}
	query := `
		SELECT object_id, count() AS cnt
		FROM stats
		WHERE event_type = ?
		GROUP BY object_id
		ORDER BY cnt DESC
		LIMIT 10
	`
	rows, err := DB.Query(query, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*protostats.TopEntry
	for rows.Next() {
		var id string
		var count int32
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		entries = append(entries, &protostats.TopEntry{
			Id:    id,
			Count: count,
		})
	}
	return &protostats.TopStats{Entries: entries}, nil
}

func (s *Server) GetTopUsers(ctx context.Context, req *protostats.TopRequest) (*protostats.TopStats, error) {
	eventType, ok := eventTypeMap(req.Metric)
	if !ok {
		return nil, fmt.Errorf("invalid metric: %s", req.Metric)
	}
	query := `
		SELECT user_id, count() AS cnt
		FROM stats
		WHERE event_type = ?
		GROUP BY user_id
		ORDER BY cnt DESC
		LIMIT 10
	`
	rows, err := DB.Query(query, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*protostats.TopEntry
	for rows.Next() {
		var id string
		var count int32
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		entries = append(entries, &protostats.TopEntry{
			Id:    id,
			Count: count,
		})
	}
	return &protostats.TopStats{Entries: entries}, nil
}

func eventTypeMap(metric string) (string, bool) {
	switch metric {
	case "view":
		return "promo_viewed", true
	case "click":
		return "promo_click", true
	case "comment":
		return "comment_published", true
	default:
		return "", false
	}
}

func main() {
	if err := InitClickhouse(); err != nil {
		log.Fatalf("ClickHouse init error: %v", err)
	}

	StartConsumer()

	lis, err := net.Listen("tcp", ":8085")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	protostats.RegisterStatsServiceServer(s, &Server{})
	log.Println("gRPC server listening on :8085")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
