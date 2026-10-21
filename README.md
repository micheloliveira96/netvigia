# NetVigia — Central de Redes

Projeto independente para desenvolvimento de uma plataforma própria de monitoramento de redes, em português do Brasil.

## Estado desta versão
Esta primeira entrega contém uma interface web responsiva de demonstração com visão geral, inventário, pesquisa e filtros, cadastro temporário, central de alertas, mapa ilustrativo, formulário de descoberta, relatórios, exportação CSV e tema claro/escuro.

**Atenção:** todos os dispositivos, alertas, gráficos e métricas atuais são fictícios. Nenhuma rede é monitorada nesta versão. O cadastro é temporário na sessão do navegador. Não existe ainda autenticação real, persistência, API, motor ICMP/SNMP ou descoberta ativa.

## Arquitetura prevista
- **Frontend:** painel web publicado na Vercel.
- **API:** serviço autenticado para gerir inventário, configurações e eventos.
- **Banco:** PostgreSQL para dispositivos, utilizadores, alertas e configurações; armazenamento de séries temporais a definir.
- **Motor Linux:** serviço contínuo para ICMP/Ping, SNMP v1/v2c/v3, agendamento e recolha de métricas.
- **Agente remoto (futuro):** recolha segura em redes distribuídas.
- **Pacote Debian:** serviços systemd, configuração, atualização e remoção controladas via `.deb`.

## Desenvolvimento local
A interface atual é um único ficheiro estático: abrir `index.html` num navegador ou servir a pasta com qualquer servidor estático.

## Roadmap
1. Modularizar o frontend em TypeScript e testes.
2. Ligar o repositório à Vercel com deploy automático.
3. Implementar autenticação e API com autorização por função.
4. Criar schema e migrações de banco.
5. Desenvolver e testar o motor ICMP e o coletor SNMP.
6. Guardar métricas e eventos reais e criar regras de alertas.
7. Empacotar o serviço Linux num `.deb` e testar instalação/atualização.

## Segurança
Não inserir segredos no frontend. Comunidades SNMP, credenciais, tokens e chaves devem ser geridos no backend, com armazenamento protegido e controlo de acesso. Não monitorar endereços sem autorização.
