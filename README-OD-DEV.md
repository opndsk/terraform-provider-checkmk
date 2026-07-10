# Terraform Provider for CheckMK - Open Desk Oy Development

## Providerin build

Tee providerin binaari projektin juuresta komennolla:

```shell
# Testaus
go build -tags checkmk_all ./...

# Build
go build -tags checkmk_all -o bin/terraform-provider-checkmk .
```

`checkmk_all`-tagi tarvitaan, koska provider käyttää CheckMK API -paketin generoitua versiokohtaista koodia. Jos haluat vain tarkistaa että koko projekti kääntyy, käytä `go build -tags checkmk_all ./...`.

## Findings

- 10.7.2026  - Korjattu: `internal/hosts/resource_host.go` ei enää lähetä tyhjää `attributes`-karttaa pelkän folder-siirron yhteydessä, jos määrittelyssä ei ole attribuuttiblokkia. Päivitys ohittaa attribuutit kokonaan, ellei niitä ole oikeasti suunnitelmassa mukana.

## 10.7.2026 - Hostin ja folderin siirtokorjaus

- Korjattu `checkmk_host`-resurssin päivityspolku niin, että `folder`-muutos käyttää Checkmk:n `actions/move/invoke`-endpointtia eikä pelkkää tavallista `UpdateHost`-kutsua
- Lisätty clientiin `MoveHost`- ja `MoveFolder`-metodit sekä polkunormalisointi, joka hyväksyy delimiterit `~`, `/` ja `\`
- Muutettu `checkmk_folder`-resurssi käyttämään folderin move-endpointtia, kun `parent` muuttuu, eikä enää pakottamaan resurssin replacea tässä tapauksessa
- Lisätty yksikkötestit hostin ja folderin move-kutsuille sekä acceptance-testit hostin folder-siirrolle ja folderin parent-siirrolle
- Varmennettu komennot `go build -tags checkmk_all ./...`, `go vet -tags checkmk_all ./...` ja `go test -tags checkmk_all -race -v -skip 'TestAcc' ./...`
- Acceptance-testejä ei ajettu tässä shellissä, koska ympäristömuuttujat `CHECKMK_URL`, `CHECKMK_USERNAME` ja `CHECKMK_PASSWORD` puuttuivat
- Komentoa `golangci-lint run` ei voitu ajaa, koska `golangci-lint`-binääri ei ollut käytettävissä `PATH`:ssa
