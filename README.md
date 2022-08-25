[![GoDoc](https://godoc.org/github.com/googleinterns/cloud-operations-api-mock?status.svg)](https://pkg.go.dev/github.com/googleinterns/cloud-operations-api-mock)

# This Repository is Archived

We recommend testing against real GCP APIs instead, or writing simple mocks for
testing your use-case locally.

# Cloud Operations API Mock

This project allows your tests to easily mock out Google Cloud Monitoring
and Google Cloud Trace.

This is not an officially supported Google product.


## Source Code Headers

Every file containing source code must include copyright and license
information. This includes any JS/CSS files that you might be serving out to
browsers. (This is to help well-intentioned people avoid accidental copying that
doesn't comply with the license.)

Apache header:

    Copyright 2020 Google LLC

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        https://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.

## Overview
Suppose you're trying to write integration tests for an exporter.

In the past, the exporter would have to point to an actual GCP instance.
  
Our mock server implements the same endpoints as the Cloud Trace and Cloud Monitoring APIs. This allows you to point the exporter to our mock server instead of calling actual GCP endpoints.

```go

mock := cloudmock.NewCloudMock()
defer mock.Shutdown()
// Get the connection info of the mock.
clientOpt := []option.ClientOption{option.WithGRPCConn(mock.ClientConn())}

// Point the exporter to the mock server.
_, flush, err := traceExporter.InstallNewPipeline(
	[]traceExporter.Option{
		traceExporter.WithTraceClientOptions(clientOpt)
	}
)

```

## Usage

All of the services can be used as a library, or as a stand-alone server

### Library Mode

Install the project:

`go get github.com/googleinterns/cloud-operations-api-mock`

Import:

` import "github.com/googleinterns/cloud-operations-api-mock/cloudmock"`

Sample usage for Cloud Trace:

```go

func TestXxx(t *testing.T) {
    // Start the mock server, defer the shutdown.
    mock := cloudmock.NewCloudMock()
    defer mock.Shutdown()
    // Set a delay of 20ms to simulate latency.
    mock.SetDelay(20 * time.Millisecond)

    // Omitted for brevity: create exporter, point to mock server and export a span.
    
    // Check that the correct number of spans was created.
    assert.EqualValues(t, 1, mock.GetNumSpans())
}

```

See the [GoDocs](https://pkg.go.dev/github.com/googleinterns/cloud-operations-api-mock@v0.0.0-20200709193332-a1e58c29bdd3/cloudmock?tab=doc) for specific functions that `cloudmock` provides.

### Stand-alone Server Mode

This mode allows the mock server to be used even if you aren't using Go. Simply start up the server and point the exporter to it.

Curl the binary:

`curl -L https://github.com/googleinterns/cloud-operations-api-mock/releases/download/[version]/mock_server-x64-linux-[version]`

Run the binary:
```
chmod +x mock_server-x64-linux-[version]
./mock_server-x64-linux-[version]
```
Optional flags: 
 
`-address=<host:port>` runs the server on the given address. If this flag is not used, a default address of `localhost:8080` is used  
`-summary` outputs a static HTML file `summary.html` when the server shuts down, which contains a summary of the data received


##### Usage in Continuous Integration:

There are many ways to use the stand-alone server in continuous integration.

An example workflow would be having CircleCI `curl` the binary and put it somewhere (Ex. `$PATH`), 
and have the test spin up the server.

Here's a simplified example from the Python Cloud Trace exporter, after the CI `curl`s the binary and adds it to `$PATH`.

```python
class SampleIntegrationTest(unittest.TestCase):
    def setUp(self):
        # Start the mock server at some address.
        args = ["mock_server-x64-linux-[version]", "-address", self.address]
        self.mock_server_process = subprocess.Popen(
            args, stderr=subprocess.PIPE
        )

    def tearDown(self):
           # Kill the mock server.
        self.mock_server_process.kill()

    def test_xxx(self):
        # Create the trace exporter and point to the address of the mock server.
        channel = grpc.insecure_channel(self.address)
        transport = trace_service_grpc_transport.TraceServiceGrpcTransport(
            channel=channel
        )
        client = TraceServiceClient(transport=transport)
        trace_exporter = CloudTraceSpanExporter(self.project_id, client=client)

        # Omitted for brevity: Create spans, export them, make assertions.
```
