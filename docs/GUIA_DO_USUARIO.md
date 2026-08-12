# Guia do usuário

## Usar online

Abra o site, cole a URL e escolha **Analisar**. Confira somente os dados retornados pelo backend, escolha qualidade e intervalo quando for playlist, leia o resumo e inicie. A página Downloads mostra estados reais. Quando concluído, use **Baixar arquivo** antes do horário de expiração.

Se aparecer “backend ainda não configurado” ou erro de indisponibilidade, o site não inventará uma conexão. Aguarde a recuperação do serviço ou use o Windows.

## Usar no Windows

Extraia o pacote e abra `ABRIR_MP3_DOWNLOADER.bat`. Não execute dentro do ZIP. O Engine abre o navegador local e salva inicialmente em `Downloads\Musicas_MP3`. Altere a pasta na tela **Configurações**; o Engine valida e persiste o caminho absoluto.

O terminal mostra o código local. Para usar o site hospedado com o Engine, copie esse código em **Configurações** no site. Não compartilhe o código com outros sites ou pessoas.

## Qualidade

VBR 0 é a recomendação. 320, 256, 192 e 128 kbps definem a codificação do MP3; escolher um bitrate alto não recupera informação ausente na fonte.

## Playlists

Deixe **De** e **Até** vazios para a playlist completa. Informe inteiros positivos para um intervalo. O cloud limita a quantidade configurada pelo operador; o Windows aceita até 500 itens por solicitação.

## Erros comuns

- **URL inválida:** use uma URL HTTP(S) oficial do YouTube.
- **Conteúdo indisponível:** confirme no navegador sem autenticação especial.
- **Engine não detectado:** abra o aplicativo Windows e confirme `/health`/Ferramentas.
- **Código inválido:** copie novamente o código mostrado pelo Engine.
- **Ferramenta indisponível:** baixe outra vez o pacote completo e valide seu hash; não desative o antivírus.
- **Arquivo expirado:** refaça o job cloud; o servidor remove resultados automaticamente.

## Direitos autorais

Não use o aplicativo para quebrar DRM, contornar paywall/autenticação, importar cookies sem consentimento ou acessar conteúdo sem autorização.
