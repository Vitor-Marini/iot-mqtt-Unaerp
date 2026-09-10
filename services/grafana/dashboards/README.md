# Dashboards

Todo arquivo `.json` deste diretório é carregado automaticamente pelo Grafana
na pasta **Estação Meteorológica** (ver `../provisioning/dashboards/dashboards.yml`).

Para versionar um dashboard criado pela interface:

1. Abra o dashboard no Grafana.
2. **Share → Export → Save to file** (marque *Export for sharing externally*).
3. Salve o `.json` aqui e faça commit.

O Grafana relê este diretório a cada 30 s — não precisa reiniciar o container.

Consultas Flux sugeridas para os painéis estão em
[`../../influxdb/README.md`](../../influxdb/README.md#consultas-úteis-flux).
