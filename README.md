# go-rakis: A DIY HTTP/1.1 Server in Go

It's a personal learning exercise – totally pointless for practical use.

## Checklist

| Feature                                                    | RFC/Documentation                                                                                                | Progress |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | -------- |
| TCP Listener & Connection Acceptance                       | [MDN](https://developer.mozilla.org/en-US/docs/Glossary/TCP_Socket)                                              | ✅        |
| Request Line Parsing (Method, Target, Version)             | [RFC 7230 §3.1.1](https://datatracker.ietf.org/doc/html/rfc7230#section-3.1.1)                                      | ✅        |
| Header Field Parsing                                       | [RFC 7230 §3.2](https://datatracker.ietf.org/doc/html/rfc7230#section-3.2)                                          | ✅        |
| Basic Routing (Path matching, Parameter extraction)        | N/A                                                                                                              | ✅        |
| `GET` Method Handling                                      | [RFC 7231 §4.3.1](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.1)                                      | ✅        |
| `HEAD` Method Handling                                     | [RFC 7231 §4.3.2](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.2)                                      | ⏳        |
| `POST` Method Handling                                     | [RFC 7231 §4.3.3](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.3)                                      | ✅        |
| `PUT` Method Handling                                      | [RFC 7231 §4.3.4](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.4)                                      | ⏳        |
| `DELETE` Method Handling                                   | [RFC 7231 §4.3.5](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.5)                                      | ⏳        |
| `CONNECT` Method Handling                                  | [RFC 7231 §4.3.6](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.6)                                      | ⏳        |
| `OPTIONS` Method Handling                                  | [RFC 7231 §4.3.7](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.7)                                      | ⏳        |
| `TRACE` Method Handling                                    | [RFC 7231 §4.3.8](https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.8)                                      | ⏳        |
| Response Status Line Generation                          | [RFC 7231 §6](https://datatracker.ietf.org/doc/html/rfc7231#section-6), [RFC 7230 §3.1.2](https://datatracker.ietf.org/doc/html/rfc7230#section-3.1.2) | ✅        |
| Response Header Generation                               | [MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers)                                                 | ✅        |
| Response Body Writing                                      | N/A                                                                                                              | ✅        |
| Gzip Response Compression (`Accept-Encoding` based)        | [RFC 7231 §5.3.4](https://datatracker.ietf.org/doc/html/rfc7231#section-5.3.4), [RFC 1952](https://datatracker.ietf.org/doc/html/rfc1952)           | ✅        |
| Concurrent Connection Handling                           | N/A                                                                                                              | ✅        |
| Request Body Parsing (`Content-Length` based)            | [RFC 7230 §3.3.2](https://datatracker.ietf.org/doc/html/rfc7230#section-3.3.2)                                      | ✅        |
| Chunked Transfer Encoding                                  | [RFC 7230 §4.1](https://datatracker.ietf.org/doc/html/rfc7230#section-4.1)                                          | ✅        |
| Persistent Connections / Keep-Alive                      | [RFC 7230 §6.3](https://datatracker.ietf.org/doc/html/rfc7230#section-6.3)                                          | ⏳        |
| Connection Timeouts                                        | [RFC 7230 §6.5](https://datatracker.ietf.org/doc/html/rfc7230#section-6.5)                                          | ⏳        |
| Content Negotiation (Accept\*, etc.)                     | [RFC 7231 §5.3](https://datatracker.ietf.org/doc/html/rfc7231#section-5.3)                                          | ⏳        |
| Caching Headers (ETag, Last-Modified, Cache-Control)     | [RFC 7232](https://datatracker.ietf.org/doc/html/rfc7232), [RFC 7234](https://datatracker.ietf.org/doc/html/rfc7234) | ⏳        |
| Conditional Requests (If-\*)                             | [RFC 7232](https://datatracker.ietf.org/doc/html/rfc7232)                                                          | ⏳        |
| Authentication (Authorization, WWW-Authenticate)           | [RFC 7235](https://datatracker.ietf.org/doc/html/rfc7235)                                                          | ⏳        |
| Range Requests (Range)                                     | [RFC 7233](https://datatracker.ietf.org/doc/html/rfc7233)                                                          | ⏳        |
| HTTPS/TLS                                                  | [RFC 2818](https://datatracker.ietf.org/doc/html/rfc2818), [RFC 8446](https://datatracker.ietf.org/doc/html/rfc8446) | ⏳        |

**Note**: This is inspired by [codecrafters.io](https://codecrafters.io)'s "Build Your Own HTTP server" challenge.

[![progress-banner](https://backend.codecrafters.io/progress/http-server/1fdacaf2-669f-433a-8cb1-4945aa9c7b6e)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

