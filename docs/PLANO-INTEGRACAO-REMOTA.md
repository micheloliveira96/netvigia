# Preparação da integração remota — NetVigia

Estado: **preparado para implementação futura; integração NÃO ativada**.

## O que já existe (confirmado no repositório)
- Serviço local Docker na porta 8787, com volume persistente e NETVIGIA_API_KEY obrigatória.
- API local documentada: GET /api/health, GET /api/devices, GET /api/metrics, POST /api/agent/metrics.
- Agentes Windows e Linux e monitorização ICMP; SNMP é opcional.
- Painel hospedado na Vercel separado do serviço local.

## Pendências antes de ligar o painel à telemetria real
1. Confirmar no ambiente Windows que o serviço Docker responde em localhost:8787 e recebe métricas recentes dos agentes; verificar logs, IDs e horários.
2. Definir transporte **de saída** do agente local para backend HTTPS, sem expor a porta 8787, o Docker socket ou a LAN. Escolher entre ingestão push autenticada e túnel privado controlado.
3. Criar endpoint de ingestão com autenticação por dispositivo, tokens rotacionáveis, limites de taxa, validação de payload, timestamps, prevenção de replay e segregação por instalação.
4. Escolher armazenamento persistente das métricas e política de retenção; não usar memória volátil de funções serverless como banco de dados.
5. Implementar endpoint de leitura autenticado no backend para o painel, com estados online, offline, desatualizado e indisponível; nunca substituir falhas por números de demonstração.
6. Configurar variáveis de ambiente na Vercel sem expor chaves no frontend; segredos reais não devem entrar no Git.
7. Adaptar UI para mostrar última atualização, origem e estado de conexão; validar no domínio de produção.
8. Router NOS: validar apenas ICMP primeiro; SNMP só se o equipamento suportar e permitir acesso seguro. Velocidade real exige fonte confiável e amostras periódicas, não inventar valores.
9. Testar falhas: agente desligado, credencial inválida, internet indisponível, dados antigos, duplicação e reinício de containers.
10. Implantar por etapas e manter rollback disponível; exigir confirmação antes de ativar transmissão de telemetria para serviço externo.

## Condições de aceite para etapa futura
- Docker, Windows e router apresentam **somente** dados reais com timestamp.
- Ausência de conexão exibe indisponível/desatualizado, nunca valores fictícios.
- Nenhuma porta administrativa da LAN é exposta publicamente.
- Chaves não aparecem no código cliente, logs ou repositório.
- Testes de autenticação, recuperação e falhas passam.

**Não executar agora:** abrir portas, configurar túnel, cadastrar segredos, alterar agentes locais, conectar banco de dados ou iniciar ingestão remota.
