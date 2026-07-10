# Terraform Provider for CheckMK - Open Desk Oy Development

## Versiointi

**Nykyinen forkin versio: `0.0.6-od.1`**

Tämä ei ole enää upstreamin alkuperäinen build, joten forkilla on oma versionumero. Skeema on `<seuraava upstream-patch>-od.<n>`, missä `n` kasvaa jokaisella Open Deskin buildilla.

Upstreamin viimeisin tagi on `v0.0.5`, joten tämä build on `0.0.6-od.1`. Älä käytä muotoa `0.0.5-od.1`, vaikka se kuulostaa loogiselta: semverissä prerelease-tunniste laskee järjestystä, joten `0.0.5-od.1` on **vanhempi** kuin `0.0.5`. Muoto `0.0.6-od.1` asettuu oikein upstream-versioiden `0.0.5` ja `0.0.6` väliin. Kun upstream julkaisee `0.0.6`:n ja se rebasetetaan tähän, seuraava versio on `0.0.7-od.1`.

Versionumero on kahdessa paikassa, ja molempien pitää vastata toisiaan:

- `main.go`:n `version`-muuttujan oletusarvo. Forkissa se on `0.0.6-od.1` eikä `dev`, jotta myös pelkkä `go build` tuottaa oikein tunnistautuvan binäärin
- Build-komennon `-ldflags "-X main.version=..."`, joka ylikirjoittaa oletuksen. Näin goreleaser tekee julkaisussa tagin perusteella

Versio ei ole pelkkä kosmeettinen tieto: se päätyy `provider.go`:n kautta sekä Terraformin `resp.Version`-kenttään että CheckMK:lle lähtevään `User-Agent`-otsakkeeseen (`client.SetProviderVersion`).

## Providerin build

Tee providerin binaari projektin juuresta komennolla:

```shell
# Testaus
go build -tags checkmk_all ./...

# Build, versioitu nimi
VERSION=0.0.6-od.1
go build -tags checkmk_all \
  -ldflags "-X main.version=${VERSION}" \
  -o "bin/terraform-provider-checkmk_v${VERSION}" .
```

`checkmk_all`-tagi tarvitaan, koska provider käyttää CheckMK API -paketin generoitua versiokohtaista koodia. Jos haluat vain tarkistaa että koko projekti kääntyy, käytä `go build -tags checkmk_all ./...`.

Tarkista buildatusta binäärista, että versio meni perille:

```shell
go version -m bin/terraform-provider-checkmk_v0.0.6-od.1 | grep -- -ldflags
```

## Binäärin nimeämiskonventio

Nimi noudattaa Terraformin konventiota `terraform-provider-checkmk_v<versio>`, eli `terraform-provider-checkmk_v0.0.6-od.1`. Tiedostonimessä on `v`-etuliite mutta itse versiomuuttujassa ei. Versio on siis pisteellistä semveriä sekä tiedostonimessä että binäärin sisällä.

Tämä nimi ei kelpaa Pulumille sellaisenaan, ja se on tietoinen valinta. `pulumi package add terraform-provider <polku>` johtaa provider-nimen suoraan tiedostonimestä poistamalla `terraform-provider-`-etuliitteen, ja tuloksen on täsmättävä Pulumi-skeeman tokenregexiin `^[a-zA-Z][-a-zA-Z0-9_]*$`. Nimestä `checkmk_v0.0.6-od.1` pisteet kaatavat komennon.

Ratkaisu tehdään vastaanottavassa repossa, ei täällä: `iac-pulumi/.providers/checkmk/`-kansioon luodaan pistevapaa symbolinen linkki, joka osoittaa versioituun binääriin, ja `pulumi package add` osoitetaan linkkiin. Versioitu tiedosto pysyy ainoana totuuslähteenä. Tästä reposta ei kirjoiteta sinne mitään.

## Build 0.0.6-od.1 (10.7.2026)

- **Binääri**: `bin/terraform-provider-checkmk_v0.0.6-od.1`
- **Lähdecommit**: `80b59cf` - "Fix: Use move endpoints for host folder and folder parent changes", plus `main.go`:n versiomuutos
- **Go-versio**: go1.26.5 (`go.mod` vaatii vähintään go1.24.1)
- **SHA256**: `5b8384402212b34b045d0ef8e084b7309065ab543c46998eb351ba0221a8865d`

Binääri kopioidaan `open-desk-infra`-repoon polkuun `iac-pulumi/.providers/checkmk/`, ja se rekisteröidään sieltä Pulumille manuaalisesti (`pulumi package add terraform-provider <polku>`). Kopioinnista ja rekisteröinnistä vastaa vastaanottava repo, ei tämä - tästä reposta ei kirjoiteta sinne mitään.

Huomio vastaanottavalle repolle: `iac-pulumi/.providers/checkmk/.gitignore` sisältää rivin `terraform-provider-checkmk`, joka **ei kata** versioitua tiedostonimeä `terraform-provider-checkmk_v0.0.6-od.1`. Ilman ignore-säännön päivitystä (esim. `terraform-provider-checkmk*`) 64 MB:n binääri päätyisi committiin. Sama sääntö kattaa myös pistevapaan symlinkin.

### Findings

- 10.7.2026 - Korjattu: `internal/hosts/resource_host.go` ei enää lähetä tyhjää `attributes`-karttaa pelkän folder-siirron yhteydessä, jos määrittelyssä ei ole attribuuttiblokkia. Päivitys ohittaa attribuutit kokonaan, ellei niitä ole oikeasti suunnitelmassa mukana.

### 10.7.2026 - Hostin ja folderin siirtokorjaus

- Korjattu `checkmk_host`-resurssin päivityspolku niin, että `folder`-muutos käyttää Checkmk:n `actions/move/invoke`-endpointtia eikä pelkkää tavallista `UpdateHost`-kutsua
- Lisätty clientiin `MoveHost`- ja `MoveFolder`-metodit sekä polkunormalisointi, joka hyväksyy delimiterit `~`, `/` ja `\`
- Muutettu `checkmk_folder`-resurssi käyttämään folderin move-endpointtia, kun `parent` muuttuu, eikä enää pakottamaan resurssin replacea tässä tapauksessa
- Lisätty yksikkötestit hostin ja folderin move-kutsuille sekä acceptance-testit hostin folder-siirrolle ja folderin parent-siirrolle
- Varmennettu komennot `go build -tags checkmk_all ./...`, `go vet -tags checkmk_all ./...` ja `go test -tags checkmk_all -race -v -skip 'TestAcc' ./...`
- Acceptance-testejä ei ajettu tässä shellissä, koska ympäristömuuttujat `CHECKMK_URL`, `CHECKMK_USERNAME` ja `CHECKMK_PASSWORD` puuttuivat
- Komentoa `golangci-lint run` ei voitu ajaa, koska `golangci-lint`-binääri ei ollut käytettävissä `PATH`:ssa
