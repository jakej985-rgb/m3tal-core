## docker/
```
old
docker/
├── media/
├── network/
├── legacy/
├── docker-compose.yml
├── docker-compose.override.yml
└── service-configs/
```
new
docker/
├── godash-compose.yml
├── media-compose.yml
├── network-compose.yml
├── routing-compose.yml
├── maintenance-compose.yml
└── .env.example
```
fix docker folder yml in docker folder root no subfolders
fix m3tal.py to auto detect all yml in the /docker folder and the /reporoot/docker folder not use the hardcode paths for each yml name so if added newone later it detects them

---

## source/
```
source/
├── dashboard/
│   ├── Dockerfile
│   ├── server.py
│   └── requirements.txt
└── go-backend/
    ├── Dockerfile
    └── main.go
```