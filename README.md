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

## Como usar

1. Baixe o projeto em **Code > Download ZIP** e extraia todos os arquivos.
2. Dê dois cliques em `Abrir_Baixador_MP3_V2.bat`.
3. Na primeira abertura, aguarde o download automático de yt-dlp, FFmpeg,
   ffprobe e Deno.
4. Escolha a opção **1** e cole a URL de um vídeo ou playlist do YouTube.

Por padrão, os arquivos são salvos em:

```text
%USERPROFILE%\Downloads\Musicas_MP3
```

O destino e as demais preferências podem ser alterados pelo menu
**Configurações**.

## Requisitos

- Windows 10 ou Windows 11 de 64 bits;
- PowerShell 5.1 ou mais recente;
- conexão com a internet;
- espaço em disco para as ferramentas e os arquivos de áudio.

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

- Use a opção **4** para atualizar e verificar as ferramentas.
- Consulte `ultimo_download.log` ou a pasta `logs` após uma falha.
- Se uma ferramenta estiver incompleta, apague o arquivo correspondente da
  pasta `ferramentas` e abra o programa novamente.

Alguns vídeos podem estar indisponíveis por restrição regional, privacidade,
autenticação ou proteção de direitos autorais.
