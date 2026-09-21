# Hetzner Control Plane

A modular, easy-to-install control plane for Hetzner Cloud that provides VM autoscaling, infrastructure orchestration, and managed service provisioning.

what this project offers:

- Managed services such as databases and caches
- VM autoscaling
- Infrastructure orchestration
- End-to-End Observability

## What is this?

An experimental cloud control plane built on top of Hetzner Cloud. It provides higher-level infrastructure and managed-service capabilities that are not available natively in Hetzner Cloud.

1. It is built with scalability in mind
2. It is modular
3. It is API-driven
4. It provides Managed service provisioning
5. It provides VM autoscaling
6. It has end-to-end observability
7. It is designed for extensibility

hetzner control plane bundles the following modules together

- api server manages Hetzner Cloud resources, templates, and autoscaling groups through REST APIs.
- control plane handles VM autoscaling by reacting to monitoring data and Grafana alerts.
- service control plane provisions and manages higher-level services such as databases and caches using automated workflows.

## Quick start

### requirements

1. a hetzner api key
2. a hetzner private network id

The easiest way to get the private network id is to create one from the console and you will find the id in the url when open it

### Installation on Hetzner

1. clone the repo

    ```bash
    git clone https://github.com/ahmedhesham301/hetzner-control-plane.git
    cd hetzner-control-plane
    ```

2. create a .env.compose file and put those values in it

    ```.env
    ENV=prod
    HKEY= put the api key here
    networkID= put the network id here
    ```

3. run

    ```bash
    docker compose up -d
    ```

### Installation outside of Hetzner

same steps in [Installation on Hetzner](#Installation on Hetzner) but replace the ENV in the .env.compose with dev
>[!WARNING]
> Exposure to the public Internet risk.
> This gives all Services a public IP and allows all incoming traffic.


## Feature plans
- object storage using s3
- managed kubernetes 
- adding more services (mysql, mongodb, redis)
- automated backups, restore, PITR, upgrades, resize/scale, credentials rotation, health checks, failover, read replicas, HA PostgreSQL with Patroni