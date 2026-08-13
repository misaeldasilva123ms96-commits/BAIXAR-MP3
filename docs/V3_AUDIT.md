# Auditoria para a plataforma V3

Data da auditoria: 12 de agosto de 2026. Base do BAIXAR-MP3: `258492c`. Referência CentralPDF: `origin/main` em `882197a`.

## Estado do BAIXAR-MP3 antes da mudança

O repositório continha nove arquivos versionados. O produto era um script PowerShell 5.1 de aproximadamente 22 KB, iniciado por BAT e executado em um menu de terminal. Não existiam frontend web, API, servidor local, pacote compilado, testes de comportamento, workflow de release ou Pages.

### Funcionalidades preservadas

- vídeo e playlist do YouTube;
- intervalo `--playlist-start`/`--playlist-end`;
- VBR 0, 320K, 256K, 192K e 128K;
- saída organizada pelo título da playlist;
- `--download-archive` para evitar repetidos;
- miniatura JPG e metadados incorporados;
- logs datados e último log;
- configuração persistente de pasta, qualidade, organização e abertura da pasta;
- preparação de yt-dlp, FFmpeg, ffprobe e Deno;
- atualização do yt-dlp e diagnóstico de versões;
- retentativas, arquivos temporários e continuação após item indisponível.

O arquivo V2 não foi refatorado. `tests/legacy-contract.ps1` protege sintaxe e opções essenciais, e o inicializador V2 continua no pacote V3.

### Riscos encontrados

- ausência de testes antes de alterações no monólito;
- download de ferramentas via URLs `latest` sem hash no fluxo V2;
- log V2 registra a URL completa;
- “sucesso” é determinado pelo exit code global mesmo com `--ignore-errors`;
- nenhuma separação entre UI, domínio e processo externo;
- nenhuma superfície segura que uma UI web pudesse chamar;
- inexistência de rate limit, TTL, cancelamento e proteção SSRF porque não havia backend.

Os riscos legados não foram ocultados. O V3 introduz limites seguros em componentes novos, mantendo o V2 como compatibilidade.

## Padrões aproveitados do CentralPDF

- Pages monta um artefato estático e mantém o deploy em job separado, desabilitado em PR;
- servidor Go pequeno, reproduzível e restrito a loopback para o pacote Windows;
- releases somente por tags pertencentes a `main`, sem substituir ativos;
- hashes e GitHub Artifact Attestations;
- pacote gerado por allowlist e verificação de arquivos pessoais proibidos;
- CI separa testes, build do app e smoke test do executável;
- documentação explícita de privacidade, segurança, limitações e solução de problemas;
- frontend com Vite/React/TypeScript e testes Vitest.

## Decisão de menor refatoração

Em vez de converter o PowerShell existente, o V3 cresce em paralelo:

1. testes de caracterização protegem o V2;
2. um core Go controla argumentos e processos reais;
3. a API cloud e o Engine reutilizam o core;
4. uma única UI usa providers HTTP tipados;
5. o release inclui V3 e o inicializador V2.

Isso permite retirada futura do V2 apenas depois de equivalência operacional comprovada em Windows real.
