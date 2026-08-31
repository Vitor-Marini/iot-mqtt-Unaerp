// Package telemetry traduz os payloads JSON de /telemetry e /health-check,
// descritos em docs/mqtt-contract.md, para tipos Go já validados.
//
// A validação de faixa (temperatura, pressão, altitude) vive aqui: um sensor
// com defeito publica -140 °C sem hesitar, e isso não deve virar série
// temporal.
package telemetry
