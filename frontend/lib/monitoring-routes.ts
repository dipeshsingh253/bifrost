export function serverPath(serverRouteID: string): string {
  return `/servers/${encodeURIComponent(serverRouteID)}`;
}

export function serverProjectsPath(serverRouteID: string): string {
  return `${serverPath(serverRouteID)}/projects`;
}

export function serverProjectPath(serverRouteID: string, projectRouteID: string): string {
  return `${serverProjectsPath(serverRouteID)}/${encodeURIComponent(projectRouteID)}`;
}

export function serverContainersPath(serverRouteID: string): string {
  return `${serverPath(serverRouteID)}/containers`;
}

export function serverContainerPath(serverRouteID: string, containerRouteID: string): string {
  return `${serverContainersPath(serverRouteID)}/${encodeURIComponent(containerRouteID)}`;
}

export function serverApiPath(serverRouteID: string): string {
  return `/api/v1/servers/${encodeURIComponent(serverRouteID)}`;
}

export function serverProjectsApiPath(serverRouteID: string): string {
  return `${serverApiPath(serverRouteID)}/projects`;
}

export function serverProjectApiPath(serverRouteID: string, projectRouteID: string): string {
  return `${serverProjectsApiPath(serverRouteID)}/${encodeURIComponent(projectRouteID)}`;
}

export function serverStandaloneContainersApiPath(serverRouteID: string): string {
  return `${serverApiPath(serverRouteID)}/containers?standalone=true`;
}

export function serverContainerApiPath(serverRouteID: string, containerRouteID: string): string {
  return `${serverApiPath(serverRouteID)}/containers/${encodeURIComponent(containerRouteID)}`;
}
