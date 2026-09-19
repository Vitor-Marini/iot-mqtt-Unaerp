import os
import docx
from docx.shared import Inches, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.table import WD_TABLE_ALIGNMENT
from docx.oxml import parse_xml
from docx.oxml.ns import nsdecls

def create_report():
    doc = docx.Document()

    # Configuração de Margens (2.5 cm / 1 polegada)
    for section in doc.sections:
        section.top_margin = Inches(1)
        section.bottom_margin = Inches(1)
        section.left_margin = Inches(1)
        section.right_margin = Inches(1)

    # Cor única para todo o texto: Preto
    COLOR_BLACK = RGBColor(0, 0, 0)

    # Estilos de Texto Base
    styles = doc.styles
    normal_style = styles['Normal']
    normal_style.font.name = 'Calibri'
    normal_style.font.size = Pt(11)
    normal_style.font.color.rgb = COLOR_BLACK
    normal_style.paragraph_format.line_spacing = 1.15
    normal_style.paragraph_format.space_after = Pt(6)

    # Título Principal
    title_p = doc.add_paragraph()
    title_p.paragraph_format.space_before = Pt(0)
    title_p.paragraph_format.space_after = Pt(4)
    title_p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run_title = title_p.add_run("RELATÓRIO DE PROJETO: ESTAÇÕES METEOROLÓGICAS IOT")
    run_title.font.name = 'Calibri'
    run_title.font.size = Pt(20)
    run_title.font.bold = True
    run_title.font.color.rgb = COLOR_BLACK

    # Subtítulo
    sub_p = doc.add_paragraph()
    sub_p.paragraph_format.space_after = Pt(12)
    sub_p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run_sub = sub_p.add_run("Monitoramento Distribuído com ESP32, MQTT, InfluxDB e Grafana")
    run_sub.font.name = 'Calibri'
    run_sub.font.size = Pt(12)
    run_sub.font.italic = True
    run_sub.font.color.rgb = COLOR_BLACK

    # Bloco de Participantes / Integrantes
    part_p = doc.add_paragraph()
    part_p.paragraph_format.space_after = Pt(4)
    part_p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run_part_title = part_p.add_run("Integrantes:")
    run_part_title.font.bold = True
    run_part_title.font.size = Pt(11)
    run_part_title.font.color.rgb = COLOR_BLACK

    participantes = [
        "Antony Felipe Lampa - 837795",
        "Felipe Granvile - 837662",
        "Gabriel Reverso Pereira - 837789",
        "Otávio Ribeiro - 838807",
        "Vitor Ferraz Marini - 837771"
    ]

    for part in participantes:
        p_item = doc.add_paragraph()
        p_item.paragraph_format.space_before = Pt(0)
        p_item.paragraph_format.space_after = Pt(2)
        p_item.alignment = WD_ALIGN_PARAGRAPH.CENTER
        run_item = p_item.add_run(part)
        run_item.font.size = Pt(10.5)
        run_item.font.color.rgb = COLOR_BLACK

    # Linha divisória horizontal preta
    p_line = doc.add_paragraph()
    p_line.paragraph_format.space_before = Pt(8)
    p_line.paragraph_format.space_after = Pt(16)
    p_line_border = parse_xml(r'<w:pBdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">'
                              r'<w:bottom w:val="single" w:sz="10" w:space="1" w:color="000000"/>'
                              r'</w:pBdr>')
    p_line._p.get_or_add_pPr().append(p_line_border)

    def add_section_heading(title):
        h = doc.add_paragraph()
        h.paragraph_format.space_before = Pt(14)
        h.paragraph_format.space_after = Pt(6)
        h.paragraph_format.keep_with_next = True
        run = h.add_run(title)
        run.font.name = 'Calibri'
        run.font.size = Pt(14)
        run.font.bold = True
        run.font.color.rgb = COLOR_BLACK
        return h

    def add_subsection_heading(title):
        h = doc.add_paragraph()
        h.paragraph_format.space_before = Pt(10)
        h.paragraph_format.space_after = Pt(4)
        h.paragraph_format.keep_with_next = True
        run = h.add_run(title)
        run.font.name = 'Calibri'
        run.font.size = Pt(12)
        run.font.bold = True
        run.font.color.rgb = COLOR_BLACK
        return h

    def set_cell_background(cell, fill_hex):
        tcPr = cell._tc.get_or_add_tcPr()
        shd = parse_xml(f'<w:shd {nsdecls("w")} w:fill="{fill_hex}"/>')
        tcPr.append(shd)

    def format_table(table, col_widths, headers, data):
        table.alignment = WD_TABLE_ALIGNMENT.CENTER
        # Header
        hdr_cells = table.rows[0].cells
        for i, header_text in enumerate(headers):
            hdr_cells[i].text = header_text
            set_cell_background(hdr_cells[i], "E5E7E9")  # Cinza claro neutro
            p = hdr_cells[i].paragraphs[0]
            p.alignment = WD_ALIGN_PARAGRAPH.CENTER
            p.paragraph_format.space_before = Pt(4)
            p.paragraph_format.space_after = Pt(4)
            for run in p.runs:
                run.font.bold = True
                run.font.color.rgb = COLOR_BLACK
                run.font.size = Pt(9.5)
        
        # Rows
        for row_idx, row_data in enumerate(data):
            row_cells = table.rows[row_idx + 1].cells
            fill = "F8F9F9" if row_idx % 2 == 1 else "FFFFFF"
            for col_idx, cell_value in enumerate(row_data):
                row_cells[col_idx].text = str(cell_value)
                set_cell_background(row_cells[col_idx], fill)
                p = row_cells[col_idx].paragraphs[0]
                p.paragraph_format.space_before = Pt(3)
                p.paragraph_format.space_after = Pt(3)
                if col_idx == 0:
                    p.alignment = WD_ALIGN_PARAGRAPH.LEFT
                else:
                    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
                for run in p.runs:
                    run.font.size = Pt(9)
                    run.font.color.rgb = COLOR_BLACK
        
        # Set widths
        for row in table.rows:
            for i, w in enumerate(col_widths):
                row.cells[i].width = Inches(w)

    # -------------------------------------------------------------
    # 1. INTRODUÇÃO
    # -------------------------------------------------------------
    add_section_heading("1. Introdução")
    doc.add_paragraph(
        "Este projeto consiste na implementação de um sistema de monitoramento climático e diagnóstico de hardware "
        "baseado em Internet das Coisas (IoT). O sistema simula três estações meteorológicas autônomas, operando com "
        "microcontroladores ESP32 e sensores barométricos BMP280 para coleta em tempo real de temperatura, pressão "
        "atmosférica e estimativa de altitude."
    )
    doc.add_paragraph(
        "Os dados coletados são transmitidos via protocolo MQTT para um broker central, ingeridos por um serviço "
        "em Go, armazenados no banco temporal InfluxDB v2 e exibidos em dashboards interativos no Grafana. Além da "
        "telemetria climática, a solução inclui monitoramento contínuo de integridade do microcontrolador (health-check) "
        "e suporte a atualizações remotas de firmware via Over-The-Air (OTA)."
    )

    # -------------------------------------------------------------
    # 2. PROBLEMA
    # -------------------------------------------------------------
    add_section_heading("2. Problema")
    doc.add_paragraph(
        "O problema central resolvido por este projeto é a necessidade de coletar, centralizar e visualizar dados "
        "climáticos de múltiplos pontos físicos distintos de forma independente, evitando a distorção gerada por médias "
        "globais que mascaram microclimas locais."
    )
    doc.add_paragraph(
        "Em conjunto, o projeto soluciona a dificuldade de manutenção e supervisão operacional de múltiplos dispositivos "
        "em campo, resolvendo:"
    )
    p_p1 = doc.add_paragraph(style='List Bullet')
    p_p1.add_run("A segregação estrita dos dados por estação, garantindo que cada ESP32 mantenha seu histórico e métricas isolados.")

    p_p2 = doc.add_paragraph(style='List Bullet')
    p_p2.add_run("A detecção de falhas de hardware, oscilações de sinal Wi-Fi e consumo excessivo de memória RAM sem necessidade de inspeção manual.")

    p_p3 = doc.add_paragraph(style='List Bullet')
    p_p3.add_run("A atualização de software de múltiplos dispositivos de forma remota (OTA), eliminando a necessidade de conexão física individual via cabo para manutenção.")

    # -------------------------------------------------------------
    # 3. ARQUITETURA
    # -------------------------------------------------------------
    add_section_heading("3. Arquitetura")
    doc.add_paragraph(
        "O sistema adota uma arquitetura em camadas orientada a eventos sob o modelo Publish/Subscribe:"
    )
    p_c1 = doc.add_paragraph(style='List Bullet')
    r = p_c1.add_run("Borda (Edge): ")
    r.bold = True
    p_c1.add_run("Três placas ESP32 efetuam a leitura periódica do sensor BMP280 (I2C) a cada 5 segundos e publicam tópicos de telemetria e health-check.")

    p_c2 = doc.add_paragraph(style='List Bullet')
    r = p_c2.add_run("Mensageria (Broker): ")
    r.bold = True
    p_c2.add_run("Eclipse Mosquitto (porta 1883) atua como intermediário assíncrono para distribuição segura das mensagens.")

    p_c3 = doc.add_paragraph(style='List Bullet')
    r = p_c3.add_run("Ingestão (Subscriber): ")
    r.bold = True
    p_c3.add_run("Microsserviço em Go que consome as publicações MQTT, valida os payloads JSON e persiste os dados no InfluxDB via Line Protocol.")

    p_c4 = doc.add_paragraph(style='List Bullet')
    r = p_c4.add_run("Armazenamento (TSDB): ")
    r.bold = True
    p_c4.add_run("InfluxDB v2.7 armazena as séries temporais no bucket 'sensors', particionadas por tags de sensor.")

    p_c5 = doc.add_paragraph(style='List Bullet')
    r = p_c5.add_run("Visualização e Gerenciamento: ")
    r.bold = True
    p_c5.add_run("Grafana 11 com consultas Flux para exibição analítica em tempo real, e servidor Go HTTP para envio de comandos e binários OTA.")

    img_arq_path = os.path.abspath("arquitetura_sistema.png")
    if os.path.exists(img_arq_path):
        p_img_arq = doc.add_paragraph()
        p_img_arq.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_img_arq.paragraph_format.space_before = Pt(6)
        p_img_arq.paragraph_format.space_after = Pt(2)
        run_img_arq = p_img_arq.add_run()
        run_img_arq.add_picture(img_arq_path, width=Inches(6.0))

        p_cap_arq = doc.add_paragraph()
        p_cap_arq.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap_arq.paragraph_format.space_after = Pt(12)
        run_cap_arq = p_cap_arq.add_run("Figura 1: Diagrama da Arquitetura do Sistema, Serviços e Topologia de Rede")
        run_cap_arq.font.size = Pt(9)
        run_cap_arq.font.italic = True
        run_cap_arq.font.color.rgb = COLOR_BLACK

    # -------------------------------------------------------------
    # 4. TECNOLOGIAS
    # -------------------------------------------------------------
    add_section_heading("4. Tecnologias")
    doc.add_paragraph(
        "As tecnologias e ferramentas empregadas no desenvolvimento do projeto compreendem:"
    )

    tech_headers = ["Camada / Componente", "Tecnologia", "Versão / Padrão", "Função"]
    tech_data = [
        ["Microcontrolador", "ESP32-WROOM-32", "Xtensa Dual-Core", "Processamento local e comunicação sem fio"],
        ["Sensor de Pressão/Temp.", "Bosch BMP280", "I2C (0x76)", "Medição de temperatura, pressão e altitude"],
        ["Ambiente de Firmware", "PlatformIO / C++", "Arduino Framework", "Código-fonte embarcado e tarefas FreeRTOS"],
        ["Broker de Mensagens", "Eclipse Mosquitto", "2.0 (MQTT 3.1.1)", "Roteamento assíncrono de tópicos"],
        ["Serviço de Ingestão", "Go (Golang)", "Go 1.22+", "Processamento concorrente e escrita no InfluxDB"],
        ["Banco de Dados", "InfluxDB", "2.7 (TSDB / Flux)", "Armazenamento temporal indexado por tags"],
        ["Painéis de Visualização", "Grafana", "11.1.0", "Dashboards em tempo real com auto-refresh"],
        ["Servidor de Atualização", "Go (Golang)", "HTTP / REST", "Distribuição de binários e orquestração OTA"],
        ["Conteinerização", "Docker & Compose", "Compose v2", "Provisionamento e isolamento de microsserviços"]
    ]
    t_tech = doc.add_table(rows=len(tech_data) + 1, cols=4)
    format_table(t_tech, [1.5, 1.4, 1.5, 2.1], tech_headers, tech_data)

    # -------------------------------------------------------------
    # 5. MODELO DE DADOS
    # -------------------------------------------------------------
    add_section_heading("5. Modelo de Dados")
    doc.add_paragraph(
        "A modelagem define os contratos de comunicação MQTT e a persistência estruturada no InfluxDB."
    )

    add_subsection_heading("5.1. Tópicos e Payloads MQTT")
    p_t1 = doc.add_paragraph(style='List Bullet')
    r = p_t1.add_run("devices/{sensor_id}/telemetry: ")
    r.bold = True
    p_t1.add_run('{"temperature": 29.01, "pressure": 946.33, "altitude": 572.7, "timestamp": 1789841151}')

    p_t2 = doc.add_paragraph(style='List Bullet')
    r = p_t2.add_run("devices/{sensor_id}/health-check: ")
    r.bold = True
    p_t2.add_run('{"status": 1, "rssi": -41, "free_heap": 209896, "uptime_ms": 579608, "timestamp": 1789841151}')

    p_t3 = doc.add_paragraph(style='List Bullet')
    r = p_t3.add_run("devices/broadcast ou devices/{sensor_id}/ota: ")
    r.bold = True
    p_t3.add_run('{"url": "http://192.168.10.130:8080/firmware/firmware.bin", "version": "1.2.0", "size": 1022384, "md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}')

    add_subsection_heading("5.2. Estrutura no InfluxDB")
    model_headers = ["Measurement", "Tags (Indexadas)", "Fields (Campos)", "Unidade"]
    model_data = [
        ["telemetry", "sensor_id, sensor_model", "temperature, pressure, altitude", "°C, hPa, m"],
        ["healthcheck", "sensor_id, sensor_model", "status, rssi, free_heap, uptime_ms", "1/0, dBm, Bytes, ms"]
    ]
    t_model = doc.add_table(rows=len(model_data) + 1, cols=4)
    format_table(t_model, [1.4, 1.8, 2.0, 1.3], model_headers, model_data)

    # -------------------------------------------------------------
    # 6. SEGURANÇA
    # -------------------------------------------------------------
    add_section_heading("6. Segurança")
    doc.add_paragraph(
        "A segurança da aplicação foi estruturada nos seguintes pilares:"
    )
    p_s1 = doc.add_paragraph(style='List Bullet')
    r = p_s1.add_run("Isolamento de Microsserviços: ")
    r.bold = True
    p_s1.add_run("Comunicação entre Subscriber, InfluxDB e Mosquitto realizada via rede interna do Docker, sem exposição externa desnecessária de portas de banco.")

    p_s2 = doc.add_paragraph(style='List Bullet')
    r = p_s2.add_run("Autenticação no InfluxDB: ")
    r.bold = True
    p_s2.add_run("Utilização de Bearer Token de 48 bytes em Base64 configurado via variáveis de ambiente seguras (.env).")

    p_s3 = doc.add_paragraph(style='List Bullet')
    r = p_s3.add_run("Integridade e Rollback no OTA: ")
    r.bold = True
    p_s3.add_run("Validação de integridade via checksum MD5 e tamanho de binário. O ESP32 utiliza esquema de partição flash dupla com rollback automático caso o novo firmware falhe na inicialização.")

    p_s4 = doc.add_paragraph(style='List Bullet')
    r = p_s4.add_run("Controle de Acesso no Grafana: ")
    r.bold = True
    p_s4.add_run("Desabilitação de autocadastro público (GF_USERS_ALLOW_SIGN_UP=false) e permissões granulares de visualização.")

    # -------------------------------------------------------------
    # 7. DASHBOARD
    # -------------------------------------------------------------
    add_section_heading("7. Dashboard")
    doc.add_paragraph(
        "Foram construídos dois dashboards no Grafana com provisionamento automático, respeitando a premissa "
        "de individualidade de cada uma das 3 estações meteorológicas:"
    )

    add_subsection_heading("7.1. Dashboard 1: Estações Meteorológicas - BMP280")
    doc.add_paragraph(
        "Apresenta cards com a última medição de Temperatura (°C), Pressão Atmosférica (hPa) e Altitude Estimada (m) "
        "por estação, gráficos de séries temporais com linhas individuais por placa e tabela comparativa consolidada."
    )

    img_estacao_path = "/tmp/screenshot_estacao.png"
    if os.path.exists(img_estacao_path):
        p_img = doc.add_paragraph()
        p_img.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_img.paragraph_format.space_before = Pt(6)
        p_img.paragraph_format.space_after = Pt(2)
        run_img = p_img.add_run()
        run_img.add_picture(img_estacao_path, width=Inches(6.0))
        
        p_cap = doc.add_paragraph()
        p_cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap.paragraph_format.space_after = Pt(12)
        run_cap = p_cap.add_run("Figura 2: Dashboard de Monitoramento Climático das Estações Meteorológicas")
        run_cap.font.size = Pt(9)
        run_cap.font.italic = True
        run_cap.font.color.rgb = COLOR_BLACK
    else:
        doc.add_paragraph("[PRINT: Dashboard Estações Meteorológicas - BMP280]")

    add_subsection_heading("7.2. Dashboard 2: Diagnóstico e Saúde dos ESP32")
    doc.add_paragraph(
        "Exibe o status de operação das estações (Online/Erro), manômetros de qualidade de sinal Wi-Fi (RSSI em dBm), "
        "monitoramento de memória RAM Heap livre (detecção de memory leaks), uptime contínuo e histórico de estabilidade."
    )

    img_health_path = "/tmp/screenshot_healthcheck.png"
    if os.path.exists(img_health_path):
        p_img2 = doc.add_paragraph()
        p_img2.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_img2.paragraph_format.space_before = Pt(6)
        p_img2.paragraph_format.space_after = Pt(2)
        run_img2 = p_img2.add_run()
        run_img2.add_picture(img_health_path, width=Inches(6.0))
        
        p_cap2 = doc.add_paragraph()
        p_cap2.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap2.paragraph_format.space_after = Pt(12)
        run_cap2 = p_cap2.add_run("Figura 3: Dashboard de Diagnóstico e Saúde dos ESP32 (Health-check)")
        run_cap2.font.size = Pt(9)
        run_cap2.font.italic = True
        run_cap2.font.color.rgb = COLOR_BLACK
    else:
        doc.add_paragraph("[PRINT: Dashboard Diagnóstico e Saúde dos ESP32]")

    # -------------------------------------------------------------
    # 8. TESTES
    # -------------------------------------------------------------
    add_section_heading("8. Testes")
    doc.add_paragraph(
        "Os testes realizados contemplaram:"
    )
    p_test1 = doc.add_paragraph(style='List Bullet')
    r = p_test1.add_run("Teste de Ingestão e Concorrência: ")
    r.bold = True
    p_test1.add_run("Validação de transmissão simultânea das três placas a cada 5 segundos sem perda de mensagens.")

    p_test2 = doc.add_paragraph(style='List Bullet')
    r = p_test2.add_run("Teste de Resiliência de Conexão: ")
    r.bold = True
    p_test2.add_run("Simulação de interrupção e restabelecimento do broker MQTT, com reconexão automática das placas sem travamentos.")

    p_test3 = doc.add_paragraph(style='List Bullet')
    r = p_test3.add_run("Teste de Atualização OTA Broadcast: ")
    r.bold = True
    p_test3.add_run("Disparo de comando OTA em 'devices/broadcast'. As três estações realizaram o download do binário v1.2.0, verificação de hash e reinicialização com sucesso.")

    p_test4 = doc.add_paragraph(style='List Bullet')
    r = p_test4.add_run("Calibração de Unidades e Nomenclatura no Grafana: ")
    r.bold = True
    p_test4.add_run("Ajuste da unidade de altitude para 'lengthm' (metros) e padronização dos rótulos para 'Estação {MAC}'.")

    # -------------------------------------------------------------
    # 9. RESULTADOS
    # -------------------------------------------------------------
    add_section_heading("9. Resultados")
    doc.add_paragraph(
        "A tabela a seguir apresenta os dados consolidados obtidos durante os testes com as três estações em operação simultânea:"
    )

    res_headers = ["Estação (MAC)", "IP Local", "FW", "Temp. (°C)", "Pressão (hPa)", "Altitude (m)", "Wi-Fi RSSI", "Heap Livre"]
    res_data = [
        ["545BA26062EC", "192.168.10.201", "v1.2.0", "29.01", "946.33", "572.7 m", "-41 dBm", "205 KiB"],
        ["688A7CC0E8FC", "192.168.10.176", "v1.2.0", "26.59", "941.93", "611.4 m", "-43 dBm", "205 KiB"],
        ["F8E843F7C630", "192.168.10.218", "v1.2.0", "34.85", "947.22", "564.9 m", "-53 dBm", "205 KiB"]
    ]
    t_res = doc.add_table(rows=len(res_data) + 1, cols=8)
    format_table(t_res, [1.1, 0.9, 0.7, 0.7, 0.8, 0.8, 0.8, 0.8], res_headers, res_data)

    doc.add_paragraph(
        "A análise dos resultados demonstrou:\n"
        "• Leituras térmicas distintas entre os nós (26.59°C a 34.85°C), validando a importância da segregação dos microclimas;\n"
        "• Estabilidade total de memória livre (~205 KiB), sem indício de vazamentos;\n"
        "• Conectividade Wi-Fi robusta (-41 a -53 dBm) com atualização em tempo real a cada 5 segundos."
    )

    # -------------------------------------------------------------
    # 10. CONCLUSÃO
    # -------------------------------------------------------------
    add_section_heading("10. Conclusão")
    doc.add_paragraph(
        "O projeto alcançou plenamente seus objetivos. A arquitetura desacoplada (MQTT, Go, InfluxDB e Grafana) "
        "comprovou sua eficácia para a telemetria distribuída de estações meteorológicas, garantindo integridade dos dados, "
        "individualidade das medições climáticas e alta confiabilidade operacional."
    )
    doc.add_paragraph(
        "A implementação dos dashboards de monitoramento e health-check, somada ao suporte de atualizações OTA seguras, "
        "fornece uma base completa e escalável, pronta para adição de novos sensores e expansão para mais estações no campo."
    )

    output_path = os.path.abspath("Relatorio_Estacoes_Meteorologicas_IoT.docx")
    doc.save(output_path)
    print(f"Relatório gerado com sucesso em: {output_path}")

if __name__ == "__main__":
    create_report()
