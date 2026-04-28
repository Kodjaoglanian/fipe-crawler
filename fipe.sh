#!/bin/bash
# FIPE Crawler API - Menu interativo

BASE="${BASE_URL:-http://localhost:8080}"

# Cores
R='\033[0;31m'
G='\033[0;32m'
Y='\033[1;33m'
B='\033[0;34m'
C='\033[0;36m'
W='\033[1;37m'
N='\033[0m'

check_deps() {
    for cmd in curl jq; do
        if ! command -v $cmd &>/dev/null; then
            echo -e "${R}Erro: $cmd não está instalado${N}"
            exit 1
        fi
    done
}

check_api() {
    if ! curl -sf "$BASE/" >/dev/null 2>&1; then
        echo -e "${R}API não está respondendo em $BASE${N}"
        echo -e "${Y}Suba com: sudo docker compose up -d${N}"
        exit 1
    fi
}

header() {
    clear
    echo -e "${C}╔═══════════════════════════════════════════════╗${N}"
    echo -e "${C}║${W}        FIPE Crawler API - Menu                ${C}║${N}"
    echo -e "${C}║${B}        $BASE                  ${C}║${N}"
    echo -e "${C}╚═══════════════════════════════════════════════╝${N}"
    echo
}

pause() {
    echo
    read -rp "Pressione ENTER para continuar..."
}

# ============================================================
# Operações
# ============================================================

op_health() {
    header
    echo -e "${Y}» Health check${N}"
    curl -s "$BASE/" | jq .
    pause
}

op_tabelas() {
    header
    echo -e "${Y}» Tabelas FIPE disponíveis (live)${N}"
    curl -s "$BASE/tabelas" | jq '.[0:20]'
    echo -e "${C}(mostrando primeiras 20)${N}"
    pause
}

op_marcas() {
    header
    echo -e "${Y}» Listar marcas${N}"
    read -rp "tabela_id: " tid
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    curl -s "$BASE/marcas?tabela_id=$tid&tipo=$tipo" | jq
    pause
}

op_modelos() {
    header
    echo -e "${Y}» Listar modelos${N}"
    read -rp "tabela_id: " tid
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    read -rp "marca_id: " mid
    curl -s "$BASE/modelos?tabela_id=$tid&tipo=$tipo&marca_id=$mid" | jq
    pause
}

op_extrair_marca() {
    header
    echo -e "${Y}» Extrair veículos de uma marca específica${N}"
    read -rp "tabela_id: " tid
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    read -rp "marca_id: " mid
    echo -e "${C}Extraindo... (pode levar minutos)${N}"
    curl -s -X POST "$BASE/extrair/veiculos" \
        -H "Content-Type: application/json" \
        -d "{\"tabela_id\":$tid,\"tipo\":$tipo,\"marca_id\":$mid}" | jq
    pause
}

op_extrair_periodo() {
    header
    echo -e "${Y}» Extrair tudo de um período (ano/mês/tipo)${N}"
    read -rp "ano (ex: 2026): " ano
    read -rp "mês (1-12): " mes
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    echo -e "${C}Extraindo... (DEMORA — pode levar horas)${N}"
    curl -s -X POST "$BASE/extrair/tudo" \
        -H "Content-Type: application/json" \
        -d "{\"ano\":\"$ano\",\"mes\":\"$mes\",\"tipo\":$tipo}" | jq
    pause
}

op_extrair_historico() {
    header
    echo -e "${Y}» Baixar TUDO (todas tabelas × todos tipos)${N}"
    echo -e "${R}AVISO: isso pode levar DEZENAS DE HORAS!${N}"
    echo
    read -rp "Limitar últimos N meses (ENTER = todos): " limite
    read -rp "Tipos (ex: '1 2 3' ou ENTER para todos): " tipos
    [ -z "$tipos" ] && tipos="1 2 3"
    echo
    read -rp "Confirma? (s/N): " conf
    [ "$conf" != "s" ] && return

    read -rp "Pausa entre requests em segundos [2]: " sleep_sec
    [ -z "$sleep_sec" ] && sleep_sec=2

    LOG="fipe_download_$(date +%Y%m%d_%H%M%S).log"
    echo -e "${C}Log: $LOG${N}"

    tabelas=$(curl -s "$BASE/tabelas")
    if [ -n "$limite" ]; then
        tabelas=$(echo "$tabelas" | jq ".[0:$limite]")
    fi
    total=$(echo "$tabelas" | jq 'length')
    count=0

    while IFS= read -r row; do
        [ -z "$row" ] && continue
        ano=$(echo "$row" | jq -r '.ano')
        mes=$(echo "$row" | jq -r '.mes' | sed 's/^0//')
        tid=$(echo "$row" | jq -r '.id')
        count=$((count+1))

        for tipo in $tipos; do
            case $tipo in
                1) tipo_nome="carro" ;;
                2) tipo_nome="moto" ;;
                3) tipo_nome="caminhão" ;;
                *) tipo_nome="?" ;;
            esac
            echo -e "${G}[$count/$total]${N} tabela=$tid $mes/$ano tipo=$tipo_nome"
            echo "[$(date +%H:%M:%S)] tabela=$tid ano=$ano mes=$mes tipo=$tipo" | tee -a "$LOG"

            attempt=0
            while [ $attempt -lt 3 ]; do
                resp=$(curl -s --max-time 7200 -X POST "$BASE/extrair/tudo" \
                    -H "Content-Type: application/json" \
                    -d "{\"ano\":\"$ano\",\"mes\":\"$mes\",\"tipo\":$tipo}")
                if echo "$resp" | jq -e '.total' >/dev/null 2>&1; then
                    echo "$resp" | jq -c | tee -a "$LOG"
                    break
                fi
                attempt=$((attempt+1))
                echo -e "${R}[falha tentativa $attempt/3]${N} $resp" | tee -a "$LOG"
                sleep $((sleep_sec * attempt * 5))
            done

            sleep "$sleep_sec"
        done
    done < <(echo "$tabelas" | jq -c '.[]')

    echo -e "${G}Concluído! Log salvo em $LOG${N}"
    pause
}

op_tabelas_salvas() {
    header
    echo -e "${Y}» Tabelas já extraídas no banco${N}"
    curl -s "$BASE/tabelas/salvas" | jq
    pause
}

op_listar_veiculos() {
    header
    echo -e "${Y}» Listar veículos salvos${N}"
    read -rp "tabela_id: " tid
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    curl -s "$BASE/veiculos?tabela_id=$tid&tipo=$tipo" | jq '.[0:10]'
    echo -e "${C}(mostrando primeiros 10)${N}"
    total=$(curl -s "$BASE/veiculos?tabela_id=$tid&tipo=$tipo" | jq 'length')
    echo -e "${G}Total: $total registros${N}"
    pause
}

op_buscar() {
    header
    echo -e "${Y}» Buscar veículos${N}"
    read -rp "termo (marca/modelo/código FIPE): " q
    curl -s "$BASE/veiculos/search?q=$(printf '%s' "$q" | jq -sRr @uri)" | jq '.[0:20]'
    pause
}

op_csv() {
    header
    echo -e "${Y}» Exportar CSV${N}"
    read -rp "tabela_id: " tid
    read -rp "tipo (1=carro 2=moto 3=caminhão): " tipo
    read -rp "arquivo de saída [fipe_${tid}_${tipo}.csv]: " arq
    [ -z "$arq" ] && arq="fipe_${tid}_${tipo}.csv"
    curl -s "$BASE/veiculos/csv?tabela_id=$tid&tipo=$tipo" -o "$arq"
    echo -e "${G}Salvo em: $arq${N}"
    ls -lh "$arq"
    pause
}

op_smoke() {
    header
    echo -e "${Y}» Smoke test (rápido)${N}"

    echo -e "\n${C}1. GET /${N}"
    curl -s "$BASE/" | jq

    echo -e "\n${C}2. GET /tabelas${N}"
    TABELAS=$(curl -s "$BASE/tabelas")
    echo "$TABELAS" | jq '.[0:3]'

    TID=$(echo "$TABELAS" | jq -r '.[0].id')
    echo -e "\n${C}3. GET /marcas?tabela_id=$TID&tipo=1${N}"
    MARCAS=$(curl -s "$BASE/marcas?tabela_id=$TID&tipo=1")
    echo "$MARCAS" | jq '.[0:3]'

    MID=$(echo "$MARCAS" | jq -r '.[0].id')
    echo -e "\n${C}4. GET /modelos?tabela_id=$TID&tipo=1&marca_id=$MID${N}"
    curl -s "$BASE/modelos?tabela_id=$TID&tipo=1&marca_id=$MID" | jq '.[0:3]'

    echo -e "\n${C}5. GET /tabelas/salvas${N}"
    curl -s "$BASE/tabelas/salvas" | jq

    echo -e "\n${G}Smoke test OK${N}"
    pause
}

op_docker_status() {
    header
    echo -e "${Y}» Status dos containers${N}"
    sudo docker compose ps
    pause
}

op_docker_logs() {
    header
    echo -e "${Y}» Logs (Ctrl+C para sair)${N}"
    sudo docker compose logs -f --tail=50
}

# ============================================================
# Menu
# ============================================================

menu() {
    while true; do
        header
        echo -e "${W}Consultas live (FIPE):${N}"
        echo "  1) Health check"
        echo "  2) Listar tabelas FIPE"
        echo "  3) Listar marcas"
        echo "  4) Listar modelos"
        echo
        echo -e "${W}Extração / persistência:${N}"
        echo "  5) Extrair veículos de uma marca"
        echo "  6) Extrair tudo de um período (ano/mês/tipo)"
        echo -e "  7) ${R}Baixar TUDO (histórico completo)${N}"
        echo
        echo -e "${W}Banco de dados:${N}"
        echo "  8) Tabelas salvas"
        echo "  9) Listar veículos salvos"
        echo " 10) Buscar veículos"
        echo " 11) Exportar CSV"
        echo
        echo -e "${W}Utilitários:${N}"
        echo " 12) Smoke test"
        echo " 13) Status containers"
        echo " 14) Logs Docker"
        echo
        echo "  0) Sair"
        echo
        read -rp "Escolha: " op

        case $op in
            1)  op_health ;;
            2)  op_tabelas ;;
            3)  op_marcas ;;
            4)  op_modelos ;;
            5)  op_extrair_marca ;;
            6)  op_extrair_periodo ;;
            7)  op_extrair_historico ;;
            8)  op_tabelas_salvas ;;
            9)  op_listar_veiculos ;;
            10) op_buscar ;;
            11) op_csv ;;
            12) op_smoke ;;
            13) op_docker_status ;;
            14) op_docker_logs ;;
            0)  exit 0 ;;
            *)  echo -e "${R}Opção inválida${N}"; sleep 1 ;;
        esac
    done
}

check_deps
check_api
menu
