package grpcratelimit

import (
	"context"
	"fmt"
	"net"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

//// UnaryServerInterceptor returns a new unary server interceptor that performs request rate limiting.
//func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
//	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
//		// TODO: implement rate limiting
//		return handler(ctx, req)
//	}
//}
//
//// StreamServerInterceptor returns a new streaming server interceptor that performs request rate limiting.
//func StreamServerInterceptor() grpc.StreamServerInterceptor {
//	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
//		// TODO: implement rate limiting
//		return handler(srv, stream)
//	}
//}
//
//// UnaryClientInterceptor returns a new unary client interceptor that performs request rate limiting.
//func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
//	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
//		// TODO: implement rate limiting
//		return invoker(ctx, method, req, reply, cc, opts...)
//	}
//}
//
//// StreamClientInterceptor returns a new streaming client interceptor that performs request rate limiting.
//func StreamClientInterceptor() grpc.StreamClientInterceptor {
//	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
//		// TODO: implement rate limiting
//		return streamer(ctx, desc, cc, method, opts...)
//	}
//}
//
//// Interceptors returns the unary and stream client interceptors.
//func Interceptors() (middleware.ClientInterceptor, middleware.ServerInterceptor) {
//	return middleware.ClientInterceptor{}, middleware.ServerInterceptor{}
//}

//type RateLimiter struct {
//	limiter *rate.Limiter
//}
//
//func NewRateLimiter(rps float64, burst int) *RateLimiter {
//	return &RateLimiter{
//		limiter: rate.NewLimiter(rate.Limit(rps), burst),
//	}
//}
//
//func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
//	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
//		if !rl.limiter.Allow() {
//			return nil, status.Errorf(
//				codes.ResourceExhausted,
//				"rate limit exceeded: %s",
//				info.FullMethod,
//			)
//		}
//		return handler(ctx, req)
//	}
//}
//
//func (rl *RateLimiter) StreamServerInterceptor() grpc.StreamServerInterceptor {
//	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
//		if !rl.limiter.Allow() {
//			return status.Errorf(
//				codes.ResourceExhausted,
//				"rate limit exceeded: %s",
//				info.FullMethod,
//			)
//		}
//		return handler(srv, ss)
//	}
//}

// IPRateLimiter stores limiters for different IP addresses
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new rate limiter manager
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// getLimiter retrieves or creates a limiter for a given IP
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
		// Optional: Add cleanup logic for old limiters
		// e.g., schedule a check to remove limiters not accessed recently
	}
	return limiter
}

// RateLimiterInterceptor provides rate limiting based on client IP
func RateLimiterInterceptor() grpc.UnaryServerInterceptor {
	// Configure the limiter (e.g., 5 requests per second with a burst of 10)
	ipLimiter := NewIPRateLimiter(rate.Limit(5), 10)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Apply rate limiting only to specific methods like Login, Register
		// Note: This requires knowing the method name, might be brittle.
		// A better approach might be to apply this selectively when building the chain or using tags.
		isLogin := info.FullMethod == "/inkMe.studio.StudioAuth/Login"          // Adjust service/method name
		isRegister := info.FullMethod == "/inkMe.studio.StudioAuth/Register"    // Adjust if service name changes
		isRefresh := info.FullMethod == "/inkMe.studio.StudioAuth/RefreshToken" // Adjust

		if isLogin || isRegister || isRefresh {
			p, ok := peer.FromContext(ctx)
			if !ok {
				// Cannot get peer info, maybe allow request or deny based on policy
				fmt.Println("Warning: Could not get peer info for rate limiting")
				// return nil, status.Error(codes.Internal, "could not identify client for rate limiting") // Option to deny
			} else {
				// Extract IP address
				var clientIP string
				if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
					clientIP = tcpAddr.IP.String()
				} else if unixAddr, ok := p.Addr.(*net.UnixAddr); ok {
					// Use a placeholder or name for Unix sockets if needed
					clientIP = "unix:" + unixAddr.Name
				} else {
					clientIP = p.Addr.String() // Fallback
				}

				if clientIP != "" {
					limiter := ipLimiter.getLimiter(clientIP)
					if !limiter.Allow() {
						// Optional: Log the rate limit event
						fmt.Printf("Rate limit exceeded for IP: %s, Method: %s\n", clientIP, info.FullMethod)
						return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded for method %s", info.FullMethod)
					}
				}
			}
		}

		// Proceed with the handler if allowed
		return handler(ctx, req)
	}
}
