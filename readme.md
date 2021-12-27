# Rocky Client

### Command Line Options
-  -cert-file string
        location of cert file
-  -key-file string
        location of key file
-  -log-path string
        Logs location (default "./logs.txt")
-  -proxy string
        Port the proxy connection takes place on. (default "localhost:9998")
-  -server string
        Server location (default "localhost:9999")
-  -target string
        Target address to forward traffic to. (default "localhost:8090")

### How to Run
#### Option 1: From source
- `make vendor`
- `make run`

#### Option 2: Build it
- `make vendor`
- `make build` - will build for current system architecture. 
- `make build-linux` - will build Linux distributable
- `make build-pi` - will build Raspberry Pi compatible distributable
- You will find the executable in the `./bin` folder.

#### Option 3: Docker
Linux images:
- `docker run -d jaeg/rocky-client:latest`

Raspberry pi images:
- `docker run -d jaeg/rocky-client:latest-pi`
