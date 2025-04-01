package main

import (
	"context"
	"log"
	"net"
	"promo"
	"time"

	"github.com/gocql/gocql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type promoServer struct {
	promo.UnimplementedPromoServiceServer
	session *gocql.Session
}

func (s *promoServer) CreatePromo(ctx context.Context, req *promo.CreatePromoRequest) (*promo.Promo, error) {
	id := gocql.TimeUUID()
	creationTime := time.Now()
	if err := s.session.Query(
		"INSERT INTO promos (id, title, description, author_id, discount_rate, promo_code, creation_date, update_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.Title, req.Description, req.AuthorId, req.DiscountRate, req.PromoCode, creationTime, creationTime,
	).Exec(); err != nil {
		return nil, err
	}
	return &promo.Promo{
		Id:           id.String(),
		Title:        req.Title,
		Description:  req.Description,
		AuthorId:     req.AuthorId,
		DiscountRate: req.DiscountRate,
		PromoCode:    req.PromoCode,
		CreationDate: timestamppb.New(creationTime),
		UpdateDate:   timestamppb.New(creationTime),
	}, nil
}

func (s *promoServer) GetPromo(ctx context.Context, req *promo.GetPromoRequest) (*promo.Promo, error) {
	var p promo.Promo
	var creationDate, updateDate time.Time
	if err := s.session.Query(
		"SELECT id, title, description, author_id, discount_rate, promo_code, creation_date, update_date FROM promos WHERE id = ? LIMIT 1",
		req.Id,
	).Scan(&p.Id, &p.Title, &p.Description, &p.AuthorId, &p.DiscountRate, &p.PromoCode, &creationDate, &updateDate); err != nil {
		return nil, err
	}
	p.CreationDate = timestamppb.New(creationDate)
	p.UpdateDate = timestamppb.New(updateDate)
	return &p, nil
}

func (s *promoServer) UpdatePromo(ctx context.Context, req *promo.UpdatePromoRequest) (*promo.Promo, error) {
	updateTime := time.Now()
	if err := s.session.Query(
		"UPDATE promos SET title = ?, description = ?, discount_rate = ?, update_date = ? WHERE id = ?",
		req.Title, req.Description, req.DiscountRate, updateTime, req.Id,
	).Exec(); err != nil {
		return nil, err
	}
	return s.GetPromo(ctx, &promo.GetPromoRequest{Id: req.Id})
}

func (s *promoServer) DeletePromo(ctx context.Context, req *promo.DeletePromoRequest) (*promo.Promo, error) {
	if err := s.session.Query("DELETE FROM promos WHERE id = ?", req.Id).Exec(); err != nil {
		return nil, err
	}
	return &promo.Promo{}, nil
}

func (s *promoServer) ListPromos(ctx context.Context, req *promo.ListPromosRequest) (*promo.ListPromosResponse, error) {
	var promos []*promo.Promo
	iter := s.session.Query("SELECT id, title, description, author_id, discount_rate, promo_code, creation_date, update_date FROM promos").Iter()
	for {
		var p promo.Promo
		var creationDate, updateDate time.Time
		if !iter.Scan(&p.Id, &p.Title, &p.Description, &p.AuthorId, &p.DiscountRate, &p.PromoCode, &creationDate, &updateDate) {
			break
		}
		p.CreationDate = timestamppb.New(creationDate)
		p.UpdateDate = timestamppb.New(updateDate)
		promos = append(promos, &p)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return &promo.ListPromosResponse{Promos: promos}, nil
}

func connectToCassandra() *gocql.Session {
	cluster := gocql.NewCluster("cassandra")
	cluster.Keyspace = "promo_service"
	cluster.Consistency = gocql.Quorum
	var session *gocql.Session
	var err error
	for i := 0; i < 10; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			log.Println("Connected to Cassandra")
			return session
		}
		log.Println("Waiting for Cassandra to be ready...")
		time.Sleep(5 * time.Second)
	}
	log.Fatal("Failed to connect to Cassandra:", err)
	return nil
}

func initializeDatabase(session *gocql.Session) {
	query := `CREATE TABLE IF NOT EXISTS promos (
		id UUID PRIMARY KEY,
		title TEXT,
		description TEXT,
		author_id UUID,
		discount_rate DOUBLE,
		promo_code TEXT,
		creation_date TIMESTAMP,
		update_date TIMESTAMP
	)`
	if err := session.Query(query).Exec(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
}

func main() {
	session := connectToCassandra()
	defer session.Close()

	initializeDatabase(session)

	server := grpc.NewServer()
	promo.RegisterPromoServiceServer(server, &promoServer{session: session})
	reflection.Register(server)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	log.Println("gRPC server started on :50051")
	if err := server.Serve(listener); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}
