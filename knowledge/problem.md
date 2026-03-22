# Self-Hosted Server & Docker Monitoring Dashboard

*(Idea for a simple, solo-dev-friendly alternative to Beszel \+ Loki/Grafana)*

## Problem

I currently run 2 VPS servers with roughly this stack on each:

- 1 PostgreSQL container  
- 1 Redis container  
- 1 Nginx container  
- service-a (docker-compose)  
  - 1 backend  
  - 1 frontend  
  - 2 workers  
- service-b (docker-compose)  
  - 1 backend  
  - 1 frontend  
  - 1 worker  
  - 1 quadrant  
- service-c (docker-compose)  
  - 2 backends  
  - 1 frontend  
  - 4 workers  
  - 2 quadrants  
- Small local LLM services (Whisper, Ollama, etc.) running directly or in containers for quick tasks

When I want to monitor these VPS instances right now, I reach for **Beszel** — and honestly I really love it.  
The UI is super clean, modern, fast, and just works without getting in the way.

But for logs I fall back to **Loki \+ Grafana**… and it feels like massive overkill for a solo dev or small team.  
The setup is complex, the learning curve is steep, and it’s clearly built for enterprise-scale environments.

I want to create a much simpler, purpose-built all-in-one monitoring solution aimed at **solo developers and small teams**.

### What the tool should let me do

1. See high-level overview of each server/VPS  
   → CPU, RAM, disk, network, load, etc.  
   (Server is the top-level entity for now)  
     
2. Click into any server to see deeper info:  
     
   - Detailed host metrics  
   - List of all running services (bare docker \+ docker-compose projects)  
   - Firewall status (ufw / firewalld / iptables)  
   - Exposed ports  
   - Active firewall rules  
   - Anything else useful about the host

   

3. Click into any service / docker-compose project to go even deeper:  
     
   - Resource usage per container (CPU, memory, IO, etc.)  
   - Container logs (tail / search)  
   - For docker-compose projects: grouped view of the whole stack  
     → e.g. see all “service-a” containers together  
     → show replica counts  
     → ports (internal \+ published)  
     → health/status  
     → container uptime, restarts, etc.  
   - Same level of detail for standalone containers (like my local Whisper/Ollama instances)

Future nice-to-have (far away):

- Simple uptime monitoring (like Uptime Kuma style — add URLs / APIs to ping)

## Desired Solution (keep it brutally simple)

Two main pieces:

1. **Lightweight agent** running on each VPS  
     
   - Can be a small Docker container **or** a systemd service  
   - Collects host metrics \+ docker info \+ (optionally) firewall/port data  
   - Sends data periodically to the central dashboard  
   - Configurable via a simple **YAML file**  
     → which docker-compose projects to include/exclude  
     → which standalone containers to monitor  
     → what host metrics to collect  
     → scrape interval, etc.  
   - Should be dead simple to install & configure

   

2. **Central dashboard** (self-hosted, single place)  
     
   - One beautiful hub that shows **all my servers**  
   - List of connected agents / VPS instances  
   - Clean drill-down navigation: Server → Services → Container / Stack details  
   - RBAC (role-based access control) — very important even for small teams  
     → owner, viewer, maybe limited editor roles later, need to have server level and as well as service level access control.  
   - Minimalist & modern UI (inspired by the simplicity of Beszel)

The dream is one single tab in my browser where I see every VPS I care about, every service stack, and I can quickly jump in to see what’s eating CPU or why something restarted — without touching SSH, without opening ten different tools.

Super clean. Super focused. No enterprise bloat.  

