package gmcredentials

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"

	"github.com/tjfoc/gmsm/x509"
	"github.com/tjfoc/gmtls"
	"github.com/tjfoc/gmtls/gmcredentials/echo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port    = ":32768"
	address = "localhost:32768"
)

var end chan bool

type server struct{}

func (s *server) Echo(ctx context.Context, req *echo.EchoRequest) (*echo.EchoResponse, error) {
	return &echo.EchoResponse{Result: req.Req}, nil
}

const ca = "testdata/caV2.pem"
const cakey = "testdata/caKeyV2.pem"

const admin = "testdata/adminV2.pem"
const adminkey = "testdata/adminKeyV2.pem"

func serverRun() {
	cert, err := gmtls.LoadX509KeyPair(ca, cakey)
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	cacert, err := ioutil.ReadFile(ca)
	if err != nil {
		log.Fatal(err)
	}
	certPool.AppendCertsFromPEM(cacert)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("fail to listen: %v", err)
	}
	creds := NewTLS(&gmtls.Config{
		ClientAuth:   gmtls.RequireAndVerifyClientCert,
		Certificates: []gmtls.Certificate{cert},
		ClientCAs:    certPool,
	})
	log.Println("111")
	s := grpc.NewServer(grpc.Creds(creds))
	echo.RegisterEchoServer(s, &server{})
	log.Println("222")
	err = s.Serve(lis)
	if err != nil {
		log.Fatalf("Serve: %v", err)
	}
	log.Println("333")
}

func clientRun() {
	cert, err := gmtls.LoadX509KeyPair(admin, adminkey)
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	cacert, err := ioutil.ReadFile(ca)
	if err != nil {
		log.Fatal(err)
	}
	certPool.AppendCertsFromPEM(cacert)
	creds := NewTLS(&gmtls.Config{
		ServerName:   "test.example.com",
		Certificates: []gmtls.Certificate{cert},
		RootCAs:      certPool,
	})
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("cannot to connect: %v", err)
	}
	defer conn.Close()
	c := echo.NewEchoClient(conn)
	echoTest(c)
	end <- true
}

func echoTest(c echo.EchoClient) {
	r, err := c.Echo(context.Background(), &echo.EchoRequest{Req: "hello"})
	if err != nil {
		log.Fatalf("failed to echo: %v", err)
	}
	fmt.Printf("%s\n", r.Result)
}

func Test(t *testing.T) {
	end = make(chan bool, 64)
	go serverRun()
	go clientRun()
	<-end
}
