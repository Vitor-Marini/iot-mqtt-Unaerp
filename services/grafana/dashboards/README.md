# Dashboards

Todo arquivo `.json` deste diretório é carregado automaticamente pelo Grafana
na pasta **Estação Meteorológica** (ver `../provisioning/dashboards/dashboards.yml`).

Para versionar um dashboard criado pela interface:

1. Abra o dashboard no Grafana.
2. **Share → Export → Save to file**, com *Export for sharing externally*
   **DESMARCADO**. Marcado, o Grafana troca todo datasource por
   `${DS_INFLUXDB}` e adiciona um bloco `__inputs` que o provisionamento por
   arquivo não resolve — os painéis sobem com
   "Datasource ${DS_INFLUXDB} was not found".
3. Salve o `.json` aqui e faça commit.

O Grafana relê este diretório a cada 30 s — não precisa reiniciar o container.

Já existe um dashboard provisionado aqui: `estacao-fleet.json`, com seletor de
dispositivo, inventário da frota e uma linha que repete por placa.

Consultas Flux sugeridas para os painéis estão em
[`../../influxdb/README.md`](../../influxdb/README.md#consultas-úteis-flux).
