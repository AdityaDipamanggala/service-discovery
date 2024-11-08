# Service-Discovery

This is simple service discovery project. There are 3 modules in this repository: 
- application
- proxy
- shared

Application is a codebase for a mocked service which when we initiated it'll self register the service to the service discovery. Proxy is a codebase to collect all the registered mocked service, and distribute all incoming request in Round Robin manners to the registered instances. Shared only act as an intermidiate package for common function or object sharing between the two others modlules.

## Installation

Steps to run: 
- Install Go in your local machine
- Git checkout the repository
- Go get all the dependency
- Run Proxy Service first by executing ```go run proxy/main.go```
- After the Proxy Service run properly, we can run the dummy application ```go run web/main.go -port=<assign port here, default 8001>```



## Usage

### Proxy Service 
Beside passing all the application request, proxy has internal endpoint to check the statistic and condition of the instances.
#### Request
Method: GET  
URL:  http://localhost:8888/stats 
#### Response
**Status Code 200**
``` json
{
    "servers": {
        "<STRING>": {
            "hit_count": "<INT>",
            "status": "<ENUM: HEALTHY, UNHEALTHY, DOWN>"
        },
        "..."
    },
    "total_hit_count": "<INT>"
}

{
    "servers": {
        "http://localhost:8081": {
            "hit_count": 19,
            "status": "DOWN"
        },
        "http://localhost:8082": {
            "hit_count": 38,
            "status": "HEALTHY"
        },
        "http://localhost:8083": {
            "hit_count": 38,
            "status": "HEALTHY"
        }
    },
    "total_hit_count": 95
}
```

### Dummy Application Service
#### Request
Method: GET  
URL: http://localhost:<PORT>/transaction   
Body:  
``` json
{
    "game": "<STRING>",
    "gamer_id": "<STRING>",
    "points": "<INT>"
}
{
    "game": "PUBGM",
    "gamer_id": "ADT123XXX",
    "points": 100
}
```

#### Response
**Status Code 200**
``` json
{
    "game": "<STRING>",
    "gamer_id": "<STRING>",
    "points": "<INT>"
}
{
    "game": "PUBGM",
    "gamer_id": "ADT123XXX",
    "points": 100
}
```
