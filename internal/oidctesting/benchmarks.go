package oidctesting

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/xenitab/go-oidc-middleware/optest"
	"github.com/xenitab/go-oidc-middleware/options"

	"github.com/stretchr/testify/require"
)

func RunBenchmarks(b *testing.B, testName string, tester tester) {
	b.Helper()

	runBenchmarkHandler(b, testName, tester)
	runBenchmarkRequirements(b, testName, tester)
	runBenchmarkHttp(b, testName, tester)
}

func runBenchmarkHandler(b *testing.B, testName string, tester tester) {
	b.Helper()

	b.Run(fmt.Sprintf("%s_handler", testName), func(b *testing.B) {
		jwtOp := optest.NewTesting(b)
		defer jwtOp.Close(b)
		opaqueOp := optest.NewTesting(b, optest.WithOpaqueAccessTokens())
		defer opaqueOp.Close(b)

		cases := []struct {
			benchDescription string
			op               *optest.OPTesting
			opts             []options.Option
		}{
			{
				benchDescription: "jwt",
				op:               jwtOp,
				opts: []options.Option{
					options.WithIssuer(jwtOp.GetURL(b)),
				},
			},
			{
				benchDescription: "opaque",
				op:               opaqueOp,
				opts: []options.Option{
					options.WithOpaqueTokensEnabled(),
					options.WithIssuer(opaqueOp.GetURL(b)),
				},
			},
		}

		for i, c := range cases {
			b.Run(c.benchDescription, func(b *testing.B) {
				b.Logf("Bench #%d: %s", i, c.benchDescription)
				handler := tester.NewHandlerFn(
					nil,
					c.opts...,
				)

				fn := func(token *optest.TokenResponse) {
					testHttpWithAuthentication(b, token, handler)
				}

				runBenchmarkConcurrent(b, c.op.GetToken, fn)
			})
		}
	})
}

func runBenchmarkRequirements(b *testing.B, testName string, tester tester) {
	b.Helper()

	b.Run(fmt.Sprintf("%s_requirements", testName), func(b *testing.B) {
		jwtOp := optest.NewTesting(b)
		defer jwtOp.Close(b)
		opaqueOp := optest.NewTesting(b, optest.WithOpaqueAccessTokens())
		defer opaqueOp.Close(b)

		cases := []struct {
			benchDescription string
			op               *optest.OPTesting
			opts             []options.Option
		}{
			{
				benchDescription: "jwt",
				op:               jwtOp,
				opts: []options.Option{
					options.WithIssuer(jwtOp.GetURL(b)),
					options.WithRequiredTokenType("JWT+AT"),
					options.WithRequiredAudience("test-client"),
				},
			},
			{
				benchDescription: "opaque",
				op:               opaqueOp,
				opts: []options.Option{
					options.WithOpaqueTokensEnabled(),
					options.WithIssuer(opaqueOp.GetURL(b)),
				},
			},
		}

		for i, c := range cases {
			b.Run(c.benchDescription, func(b *testing.B) {
				b.Logf("Bench #%d: %s", i, c.benchDescription)
				handler := tester.NewHandlerFn(
					func(claims *TestClaims) error {
						return testClaimsValueEq(claims, "sub", "test")
					},
					c.opts...,
				)

				fn := func(token *optest.TokenResponse) {
					testHttpWithAuthentication(b, token, handler)
				}

				runBenchmarkConcurrent(b, c.op.GetToken, fn)
			})
		}
	})
}

func runBenchmarkHttp(b *testing.B, testName string, tester tester) {
	b.Helper()

	b.Run(fmt.Sprintf("%s_http", testName), func(b *testing.B) {
		jwtOp := optest.NewTesting(b)
		defer jwtOp.Close(b)
		opaqueOp := optest.NewTesting(b, optest.WithOpaqueAccessTokens())
		defer opaqueOp.Close(b)

		cases := []struct {
			benchDescription string
			op               *optest.OPTesting
			opts             []options.Option
		}{
			{
				benchDescription: "jwt",
				op:               jwtOp,
				opts: []options.Option{
					options.WithIssuer(jwtOp.GetURL(b)),
				},
			},
			{
				benchDescription: "opaque",
				op:               opaqueOp,
				opts: []options.Option{
					options.WithOpaqueTokensEnabled(),
					options.WithIssuer(opaqueOp.GetURL(b)),
				},
			},
		}

		for i, c := range cases {
			b.Run(c.benchDescription, func(b *testing.B) {
				b.Logf("Bench #%d: %s", i, c.benchDescription)
				testServer := tester.NewTestServer(
					c.opts...,
				)

				defer testServer.Close()

				fn := func(token *optest.TokenResponse) {
					benchmarkHttpRequest(b, testServer.URL(), token)
				}

				runBenchmarkConcurrent(b, c.op.GetToken, fn)
			})
		}
	})
}

func runBenchmarkConcurrent(b *testing.B, getToken func(t testing.TB) *optest.TokenResponse, fn func(token *optest.TokenResponse)) {
	b.Helper()

	concurrencyLevels := []int{10}
	for _, clients := range concurrencyLevels {
		numClients := clients
		b.Run(fmt.Sprintf("%d_clients", numClients), func(b *testing.B) {
			var tokens []*optest.TokenResponse
			for i := 0; i < b.N; i++ {
				tokens = append(tokens, getToken(b))
			}

			b.ResetTimer()

			var wg sync.WaitGroup
			ch := make(chan int, numClients)
			for i := 0; i < b.N; i++ {
				token := tokens[i]
				wg.Add(1)
				ch <- 1
				go func() {
					defer wg.Done()
					fn(token)
					<-ch
				}()
			}
			wg.Wait()
		})
	}
}

func benchmarkHttpRequest(tb testing.TB, urlString string, token *optest.TokenResponse) {
	tb.Helper()

	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	require.NoError(tb, err)
	token.SetAuthHeader(req)
	res, err := http.DefaultClient.Do(req)
	require.NoError(tb, err)

	defer require.NoError(tb, res.Body.Close())

	require.Equal(tb, http.StatusOK, res.StatusCode)
}
