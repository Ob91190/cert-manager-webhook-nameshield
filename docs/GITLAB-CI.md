# GitLab CI/CD Configuration

Ce document explique comment configurer et utiliser le pipeline GitLab CI/CD pour ce projet.

## 🏗️ Pipeline Overview

Le pipeline GitLab CI/CD comprend les étapes suivantes :

1. **Test** - Tests unitaires Go et validation du code
2. **Build** - Construction et push de l'image Docker
3. **Deploy** - Déploiement automatique (optionnel)

## 🔧 Configuration Required

### Variables GitLab CI/CD

Configurez ces variables dans GitLab → Settings → CI/CD → Variables :

| Variable | Description | Exemple |
|----------|-------------|---------|
| `CI_REGISTRY_USER` | Username pour la registry Docker | `gitlab-ci-token` |
| `CI_REGISTRY_PASSWORD` | Password/Token pour la registry | `glpat-xxxxxxxxxxxx` |

### Configuration de la Registry

La registry est configurée dans `.gitlab-ci.yml` :
```yaml
REGISTRY: "registry.antemeta.io"
IMAGE_NAME: "$REGISTRY/devops/cert-manager-webhook-nameshield"
```

## 📦 Images Docker Produites

### Stratégie de Tagging

| Branche/Condition | Tag(s) Produit(s) | Exemple |
|-------------------|-------------------|---------|
| `master`/`main` | `latest`, `<commit-sha>` | `latest`, `abc123` |
| Tag Git | `<tag-name>`, `latest` | `v1.0.0`, `latest` |
| Branche feature | `<branch>-<short-sha>` | `feature-auth-abc123` |
| Merge Request | `mr-<mr-id>-<short-sha>` | `mr-42-abc123` |

### Exemples d'Images

```bash
# Images de production
registry.antemeta.io/devops/cert-manager-webhook-nameshield:latest
registry.antemeta.io/devops/cert-manager-webhook-nameshield:v1.0.0
registry.antemeta.io/devops/cert-manager-webhook-nameshield:abc123def

# Images de développement
registry.antemeta.io/devops/cert-manager-webhook-nameshield:feature-auth-abc123
registry.antemeta.io/devops/cert-manager-webhook-nameshield:mr-42-abc123
```

## 🚀 Déclenchement du Pipeline

### Automatic Triggers

- **Push sur master/main** → Build et tag `latest`
- **Push de tag** → Build et tag avec le nom du tag
- **Push sur branche** → Build avec nom de branche
- **Merge Request** → Build avec identifiant MR

### Manual Triggers

- **Deploy Staging** → Déploiement manuel vers staging
- **Deploy Production** → Déploiement manuel vers production (tags uniquement)

## 🛠️ Développement Local

### Build Local

```bash
# Utiliser le Makefile
make docker-build

# Ou utiliser le script
./scripts/build-docker.sh

# Ou directement avec Docker
docker build -t registry.antemeta.io/devops/cert-manager-webhook-nameshield:local .
```

### Test Local

```bash
# Tester l'image
make docker-test

# Ou directement
docker run --rm registry.antemeta.io/devops/cert-manager-webhook-nameshield:local --help
```

## 🔍 Monitoring et Debug

### Logs du Pipeline

1. Aller dans GitLab → CI/CD → Pipelines
2. Cliquer sur le pipeline concerné
3. Cliquer sur le job pour voir les logs

### Debug Common Issues

#### Authentication Failed
```bash
# Vérifier les variables CI/CD
echo $CI_REGISTRY_USER
echo $CI_REGISTRY_PASSWORD
```

#### Build Failed
```bash
# Vérifier le Dockerfile localement
docker build -t test-image .
```

#### Push Failed
```bash
# Tester l'accès à la registry
docker login registry.antemeta.io
```

## 📋 Maintenance

### Mise à jour des Versions

1. **Go Version** : Modifier dans `Dockerfile` et `.gitlab-ci.yml`
2. **Alpine Version** : Modifier dans `Dockerfile`
3. **Docker Version** : Modifier dans `.gitlab-ci.yml`

### Nettoyage des Images

Les images de développement doivent être nettoyées régulièrement :

```bash
# Sur la registry, configurer une politique de rétention
# Garder les 10 dernières images de chaque branche
# Supprimer les images MR après 7 jours
```

## 🔐 Sécurité

### Best Practices

1. **Secrets** : Utiliser GitLab Variables (masked + protected)
2. **Registry Access** : Limiter les permissions aux projets nécessaires
3. **Image Scanning** : Activé dans le pipeline (stage `security:scan`)
4. **Dependencies** : Mise à jour régulière des dépendances

### Security Scanning

Le pipeline inclut un scan de sécurité optionnel. Pour l'activer avec Trivy :

```yaml
# Dans .gitlab-ci.yml, décommenter dans security:scan
- docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image $IMAGE_NAME:scan-$CI_COMMIT_SHORT_SHA
```

## 📞 Support

En cas de problème avec le pipeline :

1. Vérifier les logs du pipeline
2. Tester la build localement
3. Vérifier les variables CI/CD
4. Contacter l'équipe DevOps
