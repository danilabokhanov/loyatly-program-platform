package main

import (
	"context"
	"fmt"
	"log"
	proto "loyaltyservice/proto"
	"net"
	"time"

	"github.com/gocql/gocql"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type promoServer struct {
	proto.UnimplementedPromoServiceServer
	session *gocql.Session
}

func (s *promoServer) CreatePromo(ctx context.Context, req *proto.CreatePromoRequest) (*proto.Promo, error) {
	id := gocql.TimeUUID()
	creationTime := time.Now()
	if err := s.session.Query(
		"INSERT INTO promos (id, title, description, author_id, discount_rate, promo_code, creation_date, update_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.Title, req.Description, req.AuthorId, req.DiscountRate, req.PromoCode, creationTime, creationTime,
	).Exec(); err != nil {
		return nil, err
	}
	return &proto.Promo{
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

func (s *promoServer) GetPromo(ctx context.Context, req *proto.GetPromoRequest) (*proto.Promo, error) {
	var p proto.Promo
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

func (s *promoServer) UpdatePromo(ctx context.Context, req *proto.UpdatePromoRequest) (*proto.Promo, error) {
	var authorId string
	err := s.session.Query("SELECT author_id FROM promos WHERE id = ?", req.Id).Scan(&authorId)
	if err != nil {
		return nil, fmt.Errorf("promo not found or database error: %v", err)
	}

	if authorId != req.AuthorId {
		return nil, fmt.Errorf("permission denied: only the author can update this promo")
	}

	updateTime := time.Now()
	if err := s.session.Query(
		"UPDATE promos SET title = ?, description = ?, discount_rate = ?, update_date = ? WHERE id = ?",
		req.Title, req.Description, req.DiscountRate, updateTime, req.Id,
	).Exec(); err != nil {
		return nil, err
	}

	return s.GetPromo(ctx, &proto.GetPromoRequest{Id: req.Id})
}

func (s *promoServer) DeletePromo(ctx context.Context, req *proto.DeletePromoRequest) (*empty.Empty, error) {
	var authorId string
	err := s.session.Query("SELECT author_id FROM promos WHERE id = ?", req.Id).Scan(&authorId)
	if err != nil {
		return nil, fmt.Errorf("promo not found or database error: %v", err)
	}

	if authorId != req.AuthorId {
		return nil, fmt.Errorf("permission denied: only the author can delete this promo")
	}

	if err := s.session.Query("DELETE FROM promos WHERE id = ?", req.Id).Exec(); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *promoServer) ListPromos(ctx context.Context, req *proto.ListPromosRequest) (*proto.ListPromosResponse, error) {
	var promos []*proto.Promo
	iter := s.session.Query("SELECT id, title, description, author_id, discount_rate, promo_code, creation_date, update_date FROM promos").Iter()
	for {
		var p proto.Promo
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
	return &proto.ListPromosResponse{Promos: promos}, nil
}

func connectToCassandra(host string, port int, keyspace string) *gocql.Session {
	cluster := gocql.NewCluster(host)
	cluster.Port = port
	cluster.Consistency = gocql.Quorum
	cluster.ConnectTimeout = 10 * time.Second
	cluster.Timeout = 10 * time.Second

	var session *gocql.Session
	var err error

	maxRetries := 10
	retryCount := 0

	for retryCount < maxRetries {
		session, err = cluster.CreateSession()
		if err == nil {
			log.Println("Connected to Cassandra")
			break
		}

		log.Println("Waiting for Cassandra to be ready...")
		time.Sleep(5 * time.Second)
		retryCount++
	}

	if err != nil {
		log.Fatal("Tries limit exceed ...")
		return nil
	}

	var count int
	err = session.Query("SELECT COUNT(*) FROM system_schema.keyspaces WHERE keyspace_name = ?", keyspace).Scan(&count)
	if err != nil {
		session.Close()
		log.Fatal("failed to check system_schema.keyspaces: %v", err)
		return nil
	}

	if count == 0 {
		log.Printf("key %s not found. Creating...", keyspace)
		createKeyspaceQuery := fmt.Sprintf(`
	  CREATE KEYSPACE %s
	  WITH replication = {
	   'class': 'SimpleStrategy',
	   'replication_factor': 1
	  }
	 `, keyspace)

		err = session.Query(createKeyspaceQuery).Exec()
		if err != nil {
			session.Close()
			log.Fatalf(": %v", err)
			return nil
		}
	}

	session.Close()

	cluster.Keyspace = keyspace

	retryCount = 0
	for retryCount < maxRetries {
		session, err = cluster.CreateSession()
		if err == nil {
			break
		}

		log.Println("Waiting for Cassandra to be ready...")
		time.Sleep(5 * time.Second)
		retryCount++
	}

	if err != nil {
		log.Fatal("Tries limit exceed ...")
		return nil
	}

	log.Println("Cassandra is ok")
	return session
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
	session := connectToCassandra("cassandra", 9042, "loyalty_service")
	defer session.Close()

	initializeDatabase(session)

	server := grpc.NewServer()
	proto.RegisterPromoServiceServer(server, &promoServer{session: session})
	reflection.Register(server)

	listener, err := net.Listen("tcp", ":8083")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	log.Println("gRPC server started on :8083")
	if err := server.Serve(listener); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}
