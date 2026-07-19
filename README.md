# BAIXAR MP3

Aplicativo portátil para Windows 10/11 que baixa vídeos ou playlists autorizadas
do YouTube e converte o áudio para MP3. A interface funciona no terminal e não
exige instalação de Python.

> Use somente conteúdo próprio, licenciado, em domínio público ou para o qual
> você tenha autorização. Respeite direitos autorais e os termos da plataforma.

## Recursos

- download de vídeo individual ou playlist;
- MP3 em VBR 0, 320, 256, 192 ou 128 kbps;
- organização automática por playlist;
- seleção de uma faixa de itens da playlist;
- metadados e miniatura incorporados;
- histórico para evitar downloads repetidos;
- logs por execução;
- atualização e diagnóstico das ferramentas pelo próprio menu.

## Requisitos

- Windows 10 ou Windows 11 de 64 bits;
- PowerShell 5.1 ou mais recente;
- conexão com a internet;
- espaço em disco para as ferramentas e os arquivos de áudio.

## Instalação passo a passo

O programa é portátil: não existe instalador e não é necessário instalar
Python.

1. Entre na página deste repositório no GitHub.
2. Clique no botão verde **Code**.
3. Clique em **Download ZIP**.
4. Quando o download terminar, clique com o botão direito no arquivo ZIP e
   escolha **Extrair Tudo**.
5. Abra a pasta extraída. Não execute o programa diretamente de dentro do ZIP.
6. Dê dois cliques em `Abrir_Baixador_MP3_V2.bat`.
7. Na primeira execução, aguarde o download automático de:
   - yt-dlp;
   - FFmpeg e ffprobe;
   - Deno.
8. Quando aparecer o menu **BAIXADOR MP3 V2**, a instalação está pronta.

As ferramentas ficam dentro da pasta `ferramentas`. Nas próximas aberturas, o
programa reutiliza os arquivos que já foram baixados.

### Se o Windows bloquear a abertura

1. Clique com o botão direito no ZIP baixado e escolha **Propriedades**.
2. Se aparecer a opção **Desbloquear**, marque-a e clique em **Aplicar**.
3. Extraia o ZIP novamente e abra `Abrir_Baixador_MP3_V2.bat`.

Não é necessário desativar o antivírus nem alterar permanentemente a política
de execução do PowerShell.

## Como baixar uma música

1. Abra `Abrir_Baixador_MP3_V2.bat`.
2. No menu principal, digite **1** e pressione **Enter**.
3. Copie a URL do vídeo no YouTube.
4. Cole a URL na janela do programa e pressione **Enter**.
5. Escolha a qualidade:
   - **1 — VBR 0:** melhor opção geral e recomendada;
   - **2 — 320 kbps:** bitrate fixo alto;
   - **3 — 256 kbps:** boa qualidade com arquivo um pouco menor;
   - **4 — 192 kbps:** equilíbrio entre qualidade e tamanho;
   - **5 — 128 kbps:** menor tamanho.
6. Quando o programa perguntar sobre pasta de playlist, pressione **Enter** para
   manter a opção padrão. Em um vídeo individual, isso não cria uma pasta
   desnecessária.
7. Deixe os campos **Começar pelo item** e **Terminar no item** vazios,
   pressionando **Enter** em cada um.
8. Confira o resumo, digite **S** e pressione **Enter**.
9. Aguarde a mensagem **Download concluído com sucesso**.

Ao terminar, a pasta das músicas é aberta automaticamente.

> Escolher 320 kbps não aumenta a qualidade do áudio original. Para a maioria
> dos casos, use **VBR 0**.

## Como baixar uma playlist

1. Abra o programa e escolha a opção **1**.
2. Cole a URL completa da playlist.
3. Escolha a qualidade desejada.
4. Responda **S** para criar uma pasta com o nome da playlist.
5. Para baixar a playlist inteira, deixe os campos inicial e final vazios.
6. Para baixar somente uma parte, informe as posições. Exemplo:
   - **Começar pelo item:** `5`
   - **Terminar no item:** `12`
7. Confira o resumo, digite **S** e aguarde a conclusão.

O programa continua quando um item isolado está indisponível e registra os
detalhes no log.

## Onde ficam as músicas

Por padrão, os arquivos são salvos em:

```text
%USERPROFILE%\Downloads\Musicas_MP3
```

Há duas formas de abrir essa pasta:

1. escolha a opção **2 — Abrir pasta de músicas** no menu; ou
2. aguarde o programa abri-la automaticamente ao final do download.

## Como configurar o programa

Escolha **3 — Configurações** no menu principal. Nessa tela é possível:

1. selecionar outra pasta de destino;
2. alterar a qualidade padrão;
3. ativar ou desativar a organização por playlist;
4. ativar ou desativar a abertura automática da pasta ao terminar.

As escolhas são gravadas localmente em `configuracao.txt`.

## Opções do menu

| Opção | Função |
| --- | --- |
| **1** | Baixar um vídeo ou playlist em MP3 |
| **2** | Abrir a pasta onde as músicas são salvas |
| **3** | Alterar pasta, qualidade e organização |
| **4** | Atualizar e verificar yt-dlp, Deno e FFmpeg |
| **5** | Abrir o log do último download |
| **6** | Limpar o histórico que impede downloads repetidos |
| **0** | Fechar o programa |

## Histórico de downloads

O arquivo `historico_downloads.txt` registra os vídeos já processados e evita
que a mesma música seja baixada novamente.

Para baixar novamente uma música já registrada:

1. escolha a opção **6**;
2. digite `LIMPAR` para confirmar;
3. inicie o download novamente.

Limpar o histórico não apaga nenhum MP3.

## Privacidade e arquivos locais

As ferramentas baixadas, preferências, URLs já processadas, logs e arquivos
temporários ficam somente no computador do usuário. Esses itens são ignorados
pelo Git e não fazem parte do repositório.

## Estrutura

- `Abrir_Baixador_MP3_V2.bat`: inicializador para dois cliques;
- `Baixador_MP3_V2.ps1`: aplicativo principal;
- `configuracao.exemplo.txt`: referência das configurações disponíveis;
- `LEIA-ME.txt`: instruções em texto simples;
- `ALTERACOES_V2.txt`: histórico da versão 2.

## Solução de problemas

### O programa não abre

- Confirme que todos os arquivos foram extraídos do ZIP.
- Tente o procedimento **Se o Windows bloquear a abertura**.
- Mantenha o arquivo `.bat` e o arquivo `.ps1` na mesma pasta.

### Uma ferramenta não foi baixada

- Confirme que há conexão com a internet.
- Feche o programa e abra novamente.
- Use a opção **4** para atualizar e verificar as ferramentas.
- Se necessário, apague somente o arquivo com problema dentro de `ferramentas`;
  ele será baixado novamente.

### O download falhou

- Confirme que a URL é de `youtube.com` ou `youtu.be`.
- Verifique se o vídeo está disponível no navegador.
- Consulte `ultimo_download.log` ou a pasta `logs`.
- Use a opção **4** e tente novamente.

Alguns vídeos podem estar indisponíveis por restrição regional, privacidade,
autenticação ou proteção de direitos autorais.
