package server

import (
	"context"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	api "godistrserv/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

const (
	objectWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

var _ api.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (
	*grpc.Server,
	error,
) {

	/*
		We hook up our authenticate() interceptor to our gRPC server so that our server
		identifies the subject of each RPC to kick off the authorization process.
	*/

	// StreamInterceptor returns a ServerOption that sets
	//the StreamServerInterceptor for the
	// server. Only one stream interceptor can be installed.

	// StreamServerInterceptor provides a hook to intercept the execution of a streaming RPC on the server.
	// info contains all the information of this RPC the interceptor can operate on. And handler is the
	// service method implementation. It is the responsibility of the interceptor to invoke handler to
	// complete the RPC.

	// ChainStreamServer creates a single interceptor out of a chain of many interceptors.

	// UnaryInterceptor returns a ServerOption that sets the UnaryServerInterceptor for the
	// server. Only one unary interceptor can be installed. The construction of multiple
	// interceptors (e.g., chaining) can be implemented at the caller.

	// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.

	// UnaryServerInterceptor provides a hook to intercept the execution of a unary RPC on the server. info
	// contains all the information of this RPC the interceptor can operate on. And handler is the wrapper
	// of the service method implementation. It is the responsibility of the interceptor to invoke handler
	// to complete the RPC.

	opts = append(opts, grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_auth.StreamServerInterceptor(authenticate),
		),
	),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_auth.UnaryServerInterceptor(authenticate),
		)))
	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil

	/*
		В качестве перехватчика используется grpc_auth.StreamServerInterceptor(authenticate) и
		grpc_auth.UnaryServerInterceptor(authenticate),
		что добавляет проверку аутентификации для всех потоковых и обычных gRPC методов.
	*/

}
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (
	*api.ProduceResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}
func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (
	*api.ConsumeResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		consumeAction,
	); err != nil {
		return nil, err
	}

	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

/*
ProduceStream (api.Log_ProduceStreamServer) implements a bidirectional
streaming RPC so the client can stream data into the
server’s log and the server can tell
the client whether each request succeeded.
*/
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

/*
ConsumeStream (*api.ConsumeRequest,
api.Log_ConsumeStreamServer) implements a server-side streaming RPC so the
client can tell the server where in the log to read records, and then the server
will stream every record that follows—even records that aren’t in the log yet!
When the server reaches the end of the log, the server will wait until someone
appends a record to the log and then continue streaming records to the client.
*/
func (s *grpcServer) ConsumeStream(
	req *api.ConsumeRequest,
	stream api.Log_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err = stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

func authenticate(ctx context.Context) (context.Context, error) {
	peerGRPC, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}
	if peerGRPC.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}
	tlsInfo := peerGRPC.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)
	return ctx, nil
}
func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
