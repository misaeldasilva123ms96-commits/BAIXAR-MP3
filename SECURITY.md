# Política de segurança

## Versões suportadas

A linha V3 recebe correções de segurança. O V2 permanece como compatibilidade e não deve ser exposto como servidor.

## Relato responsável

Use o recurso privado **Report a vulnerability** do GitHub Security Advisories deste repositório. Não inclua tokens, cookies, URLs pessoais, MP3 ou dados de terceiros. Não abra issue pública antes da correção coordenada.

## Modelo de segurança

- somente URLs HTTP(S) de hosts oficiais conhecidos são aceitas;
- localhost, IPs privados/reservados, credenciais em URL, `file://` e `ftp://` são rejeitados;
- redirects e alterações de comportamento do extrator devem ser reavaliados a cada atualização do yt-dlp;
- a API converte campos tipados em uma allowlist interna de argumentos;
- o Engine escuta exclusivamente em `127.0.0.1:38765`, restringe CORS e exige token para POST/DELETE;
- cloud usa fila, rate limit por IP, timeout, limite de playlist, diretório por job e TTL;
- resultados cloud são temporários e não formam biblioteca;
- ferramentas do pacote são obtidas por URL versionada e SHA-256 fixo;
- releases possuem checksums e atestação de proveniência.

## Limitações

CORS não substitui autenticação; por isso o token local é obrigatório. O token é uma capability e deve ser protegido. O executável ainda não possui assinatura Authenticode. Rate limiting em memória deve ser complementado pelo proxy quando houver várias réplicas.

O aplicativo não implementa cookies de navegador, DRM, login, paywall, sessão importada nem contorno de controles de acesso.
