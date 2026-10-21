# NetVigia — Central de Monitoramento

NetVigia é uma implementação independente. A documentação pública do Monsta foi consultada para observar padrões funcionais (inventário, monitores automáticos, SNMP/ICMP e agentes); nenhum código ou recurso proprietário do Monsta é reutilizado.

## Estado do MVP 0.1.0
- Serviço Go para Linux com API REST protegida por chave e persistência local em JSON.
- Inventário de equipamentos, verificação ICMP/Ping e histórico de métricas.
- Agendamento padrão a cada 60 minutos, configurável.
- SNMP opcional através de snmpget do sistema para uptime e contadores IF-MIB, se configurado o índice da interface.
- Endpoint para ingestão de métricas de agentes Windows/Linux.
- Painel estático servido pelo serviço e script para gerar pacote Debian amd64.

**Limites honestos:** MVP inicial; ainda não tem autenticação multiutilizador, TLS próprio, PostgreSQL, gráficos alimentados por métricas reais, descoberta ativa de rede, SNMPv3 validado, cálculo de banda em bits/s por deltas, nem teste de instalação end-to-end. O instalador gera uma chave de API aleatória e o serviço escuta na interface configurada. Não exponha a porta 8787 diretamente à Internet: limite-a à LAN por firewall ou use VPN e HTTPS reverse proxy. Guarde a chave fora de repositórios e partilhas.

## Compilar e executar em desenvolvimento
Requisitos: Go 1.22+, Linux e ping. Para SNMP, instale o pacote snmp.

    cd agent
    go build -o netvigia .
    NETVIGIA_CONFIG=../packaging/config.json ./netvigia

Abra http://127.0.0.1:8787. Configure endereço de escuta e chave em packaging/config.json ou /etc/netvigia/config.json.

## Criar pacote .deb
Em Debian/Ubuntu:

    sudo apt-get update
    sudo apt-get install -y golang-go dpkg-dev snmp iputils-ping
    ./scripts/build-deb.sh 0.1.0 amd64
    sudo dpkg -i build/netvigia_0.1.0_amd64.deb
    sudo nano /etc/netvigia/config.json
    sudo systemctl restart netvigia
    sudo systemctl status netvigia --no-pager

O pacote gerado fica em build/netvigia_0.1.0_amd64.deb. A compilação precisa de ambiente Linux Debian/Ubuntu; GitHub/Vercel ainda não geram um pacote testado automaticamente.

## Teste local com Docker Desktop (Windows)
O repositório também inclui Dockerfile e docker-compose.yml para experimentar o serviço no computador que já usa Docker Desktop.

    git clone https://github.com/micheloliveira96/netvigia.git
    cd netvigia
    copy .env.example .env

Edite o arquivo .env e troque NETVIGIA_API_KEY por uma chave aleatória longa. Depois execute:

    docker compose up -d --build
    docker compose logs -f netvigia

Abra http://localhost:8787. Para conectar o agente Windows, use a URL http://IP-DA-MAQUINA-WINDOWS:8787 se o agente estiver em outra máquina; para a própria máquina, localhost pode ser usado. No Windows, veja o IP LAN com ipconfig. Não encaminhe a porta 8787 no router para a Internet. A rede bridge do Docker Desktop consegue alcançar muitos roteadores LAN, mas isso deve ser validado no ambiente real.

## API
- GET /api/health — sem chave.
- GET /api/devices — com X-API-Key ou Authorization: Bearer <chave>.
- POST /api/devices — exemplo JSON: {"name":"Roteador TP-Link","address":"192.168.1.1","type":"router","interval_minutes":60,"snmp_community":"SOMENTE-LOCAL","interface_index":1}.
- GET /api/metrics?device_id=dev-...
- POST /api/agent/metrics — exemplo: {"device_id":"dev-...","name":"PC-WINDOWS","metrics":{"cpu_percent":12.4,"memory_used_percent":54.2}}.
- POST /api/check-now — dispara verificações manuais.

## Como ligar os equipamentos
### Roteador TP-Link
1. Descubra o IP LAN no Windows com ipconfig; o Gateway predefinido costuma ser o roteador.
2. Consulte o manual do modelo/firmware: nem todos os TP-Link suportam SNMP. Se houver opção, ative SNMPv2c, restrinja o gestor ao IP do NetVigia e use comunidade de leitura forte. Nunca exponha SNMP à Internet.
3. Cadastre o IP: Ping determina online/offline. SNMP pode trazer uptime e octetos da interface se snmp estiver instalado e o índice da interface for conhecido.
4. Para mostrar velocidade de download/upload em bits/s, é necessário recolher contadores em dois instantes e calcular a diferença por segundo. O MVP ainda só guarda contadores, não calcula a taxa final.

### Computador Windows
1. Cadastre o PC com tipo windows e o endereço/IP.
2. Edite agents/windows/netvigia-agent.ps1 com URL do servidor, ID do equipamento e chave.
3. Agende no Task Scheduler (por exemplo, a cada 5 minutos) executando PowerShell. O servidor precisa ser acessível pela LAN/VPN; libere a porta 8787 apenas para redes confiáveis.
4. O script envia CPU, memória, espaço livre em disco e uptime. O intervalo do agente é definido no Agendador de Tarefas; por exemplo, a cada 5 minutos.

### Servidor Docker/Linux
- Instale NetVigia no host Linux para monitorar a disponibilidade dos equipamentos e servir o painel local.
- O script agents/linux/netvigia-agent.sh envia métricas básicas do host e quantidade de containers ativos. Acesso ao socket Docker equivale a privilégios elevados; avalie esse risco antes de conceder permissão.
- Para métricas detalhadas por container, a próxima etapa é integrar Docker Engine API.

## Interface online e rede privada
A Vercel hospeda o painel estático, mas o navegador no site não alcança automaticamente uma API privada na LAN. Para operar remotamente, publique a API atrás de HTTPS e autenticação forte via VPN/reverse proxy; não crie port-forward aberto para a porta 8787. O pacote .deb também serve o painel local em http://IP-DO-SERVIDOR:8787.
