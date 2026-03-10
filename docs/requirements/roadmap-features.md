# Feature Evaluator - Requerimientos de Nuevas Funcionalidades

Documento de requerimientos para las funcionalidades necesarias para operar independientemente de FeatBit.

## Estado actual

Al 2026-03-06, el repo ya implementa las cinco iniciativas principales, pero no todas quedaron exactamente con el alcance original de roadmap. Este documento ya no debe leerse como backlog puro, sino como estado real:

- `Percentage Rollouts`: implementado en backend y consola.
- `Historial de Cambios`: implementado y visible en consola.
- `Scheduled Rollouts`: implementado parcialmente; hoy cubre cambios de feature, pero no rollout porcentual de reglas.
- `A/B Testing`: implementado; queda deuda menor de endurecer algunos límites de producto en validación.
- `Multi-Workspace`: implementado con fallback a `default`; sigue pendiente la restricción explícita de "no eliminar si tiene datos", porque hoy `DELETE` archiva.

Las casillas marcadas como pendientes representan gap funcional real o deuda de verificación explícita, no trabajo ya presente en código.

---

## 1. Percentage Rollouts (Liberacion Gradual)

### Que se necesita

Poder liberar una feature gradualmente a un porcentaje de usuarios en lugar de activarla para todos o nadie. Por ejemplo: "habilitar para el 10% de los usuarios, luego subir a 25%, luego 50%, luego 100%".

### Por que

- **Reducir riesgo**: Si una feature tiene un bug, solo afecta al porcentaje expuesto, no a toda la base de usuarios.
- **Canary releases**: Validar que una feature funciona correctamente en produccion con un grupo pequeno antes de llegar a todos.
- **Rollback instantaneo**: Si algo falla, bajar el porcentaje a 0% sin necesidad de deploy.

### Que se desea lograr

- Configurar un porcentaje (0-100%) en cada regla de una feature.
- Que la asignacion sea **determinista**: el mismo usuario siempre recibe el mismo resultado (no cambia entre requests).
- Que sea **monotonica**: si subo de 10% a 20%, los usuarios que ya estaban en el 10% siguen incluidos.
- Que el porcentaje sea configurable desde la consola con un control visual (slider).
- Que sea compatible con las reglas existentes: el porcentaje se aplica despues de evaluar la expresion de la regla.

### Estado actual

Implementado. El backend aplica bucketing deterministico y monotonicidad con FNV-1a, y la consola expone slider e indicador visual por regla.

### Criterios de Aceptacion

- [x] Una regla con 50% aplica a aproximadamente la mitad de los usuarios.
- [x] El mismo usuario siempre obtiene el mismo resultado para la misma feature.
- [x] Incrementar el porcentaje no excluye a usuarios que ya estaban incluidos.
- [x] Sin porcentaje configurado, la regla se comporta como hoy (aplica a todos).
- [x] La consola permite configurar el porcentaje y lo muestra en el listado de reglas.

---

## 2. Historial de Cambios

### Que se necesita

Registrar automaticamente cada cambio realizado a features, reglas, segmentos y packs, incluyendo quien hizo el cambio, cuando, y que cambio exactamente.

### Por que

- **Auditoria**: Poder responder "quien habilito esta feature en produccion?" o "que cambio causo este comportamiento inesperado?".
- **Debugging**: Cuando algo se rompe, necesitamos ver que cambio inmediatamente antes.
- **Compliance**: Tener un registro inmutable de todas las modificaciones para auditoria interna.
- **Confianza**: Los equipos pueden hacer cambios sabiendo que pueden rastrear y entender cualquier modificacion.

Hoy solo existe un log de errores de evaluacion, pero no hay registro de cambios administrativos.

### Que se desea lograr

- Cada vez que alguien crea, edita, elimina o habilita/deshabilita una feature (o regla, segmento, pack), se registra automaticamente.
- El registro incluye: que entidad cambio, que campos se modificaron (valor anterior y nuevo), quien lo hizo, y cuando.
- Se puede consultar el historial completo de una feature y ver una linea de tiempo de todos sus cambios.
- Se puede ver el "diff" entre el estado anterior y posterior de un cambio.
- Se puede filtrar el historial por usuario, tipo de entidad, accion y rango de fechas.
- Los registros son inmutables (no se pueden editar ni borrar).

### Estado actual

Implementado. Existe `changelog` inmutable en backend, escritura fire-and-forget y vistas en consola para timeline global y timeline por feature.

### Criterios de Aceptacion

- [x] Toda operacion de creacion, edicion, eliminacion y toggle genera un registro de cambio.
- [x] El registro muestra exactamente que campos cambiaron, con valor anterior y nuevo.
- [x] Se puede ver el historial de una feature especifica en la consola.
- [x] Se puede filtrar por quien hizo el cambio, tipo de accion y rango de fechas.
- [x] Los registros no se pueden modificar ni eliminar.
- [x] El registro de cambios no impacta notablemente el rendimiento de las operaciones normales.

---

## 3. Scheduled Rollouts (Cambios Programados)

### Que se necesita

Poder programar cambios futuros en features que se apliquen automaticamente a una fecha y hora especifica. Por ejemplo: "habilitar feature X el lunes a las 9:00 AM" o "subir el porcentaje de rollout al 50% el viernes a las 6:00 PM".

### Por que

- **Lanzamientos coordinados**: Marketing anuncia una feature el lunes, y queremos que se active exactamente a esa hora sin intervencion manual.
- **Cambios fuera de horario**: Programar cambios para la madrugada o fin de semana sin que alguien tenga que estar pendiente.
- **Rollouts graduales automaticos**: Incrementar el porcentaje de rollout automaticamente en intervalos programados.

Hoy existen campos `activeFrom`/`activeUntil` que controlan ventanas de disponibilidad, pero no permiten programar cambios de configuracion (como cambiar el porcentaje de rollout o modificar reglas).

### Que se desea lograr

- Programar la habilitacion/deshabilitacion de una feature para una fecha y hora futura.
- Programar cambios de configuracion de feature para una fecha futura.
- Ver los cambios programados pendientes en el detalle de una feature con un countdown.
- Poder cancelar un cambio programado antes de que se ejecute.
- Los cambios ejecutados quedan registrados en el historial de cambios (integracion con feature 2).
- Si hay multiples instancias del servicio, el cambio se ejecuta una sola vez (sin duplicados).

### Estado actual

Implementado parcialmente. Hoy el scheduler soporta `toggle`, `default_value`, `environment` y `update` de feature, con worker cada 30 segundos, cancelacion y registro en changelog. No existe aun soporte especifico para programar el porcentaje de rollout de una regla.

### Criterios de Aceptacion

- [x] Se puede programar un toggle de feature para una fecha y hora futura.
- [ ] Se puede programar un cambio de porcentaje de rollout para una fecha futura.
- [x] El cambio se ejecuta automaticamente dentro de 1 minuto de la hora programada.
- [x] Un cambio cancelado no se ejecuta.
- [x] Los cambios ejecutados aparecen en el historial de cambios.
- [x] No hay ejecuciones duplicadas cuando el servicio tiene multiples instancias.
- [x] La consola muestra los cambios pendientes y permite cancelarlos.

### Deuda abierta

- Agregar scheduling de rollout porcentual a nivel de regla, si ese alcance sigue siendo deseado.

---

## 4. A/B Testing y Experimentacion

### Que se necesita

Un sistema para ejecutar experimentos controlados: dividir usuarios en grupos (control vs variantes), trackear metricas de resultado (conversiones, clicks, revenue, etc.), y determinar cual variante es mejor con evidencia estadistica.

### Por que

- **Decisiones basadas en datos**: En lugar de lanzar una feature porque "creemos que es mejor", medimos su impacto real.
- **Medir impacto**: Saber si un cambio en el flujo de onboarding realmente mejora la conversion, o si un nuevo diseno de boton aumenta los clicks.
- **Reducir riesgo de producto**: Antes de hacer rollout completo de una feature, validar que tiene el efecto deseado.
- **Cultura de experimentacion**: Dar al equipo las herramientas para proponer, medir y decidir basado en evidencia.

Hoy se podria simular un A/B test con reglas manuales, pero no hay forma de trackear resultados, medir conversiones ni calcular si la diferencia es estadisticamente significativa.

### Que se desea lograr

- Crear un experimento asociado a una feature, definiendo variantes (ej: control, variante A, variante B) con pesos de distribucion.
- Definir metricas de exito a trackear (ej: "conversion", "revenue", "tiempo en pagina").
- Al evaluar una feature con experimento activo, asignar al usuario a una variante de forma determinista y registrar la exposicion automaticamente.
- Permitir que los clientes reporten eventos de conversion (ej: "el usuario completo el onboarding").
- Ver resultados en tiempo real: exposiciones, conversiones, tasa de conversion por variante, e indicador de significancia estadistica.
- Declarar un ganador cuando hay suficiente evidencia estadistica, y aplicar su valor como el default de la feature.

### Estado actual

Implementado. Existe CRUD y lifecycle de experimentos, asignacion determinista, registro de exposures, endpoint de conversiones, resultados con intervalos de confianza y declaracion manual de ganador. El override de experimento activo sobre rollout/reglas ya existe en el pipeline.

### Criterios de Aceptacion

- [x] Se puede crear un experimento con 2 variantes o mas y pesos que sumen 100%.
- [x] La asignacion de variante es determinista (mismo usuario = misma variante siempre).
- [x] La evaluacion de una feature con experimento activo retorna la variante asignada e indica que es parte de un experimento.
- [x] Las exposiciones se registran automaticamente al evaluar.
- [x] Se pueden reportar eventos de conversion desde los clientes.
- [x] Los resultados muestran tasa de conversion e intervalo de confianza por variante.
- [x] Se puede declarar un ganador, lo cual aplica su valor como default de la feature.
- [x] Una feature solo puede tener un experimento activo a la vez.

### Deuda abierta

- El roadmap original hablaba de 2 a 4 variantes. Hoy backend y UI exigen minimo 2 y suma de pesos 100, pero no hay una restriccion dura de maximo 4 variantes.

---

## 5. Multi-Workspace

### Que se necesita

Permitir crear multiples espacios de trabajo (workspaces) aislados dentro de la misma instancia. Cada workspace tiene sus propias features, segmentos, packs, miembros y API keys. Los datos de un workspace no son visibles ni accesibles desde otro.

### Por que

- **Aislamiento por equipo/producto**: El equipo de CRM gestiona sus flags sin ver los del equipo de Educacion, y viceversa. Evita confusion y errores accidentales.
- **Permisos diferenciados**: Un usuario puede ser admin en un workspace y viewer en otro, segun su rol en cada equipo.
- **Organizacion**: A medida que crece la cantidad de features y equipos, tener todo en un unico espacio se vuelve inmanejable.

Hoy todos los datos viven en un namespace global unico. Todos los miembros ven todas las features, segmentos y packs sin distincion.

### Que se desea lograr

- Crear y gestionar multiples workspaces, cada uno con su nombre y miembros.
- Que features, segmentos, packs, tags y API keys pertenezcan a un workspace especifico.
- Que un usuario pueda pertenecer a multiples workspaces con roles diferentes en cada uno.
- Que las API keys de evaluacion solo puedan evaluar features de su workspace.
- Que la consola tenga un selector de workspace para cambiar entre ellos.
- Que los feature keys puedan repetirse entre workspaces (ej: workspace A y B pueden tener ambos un flag `dark-mode`).
- Que los datos existentes migren automaticamente a un workspace "default" para no romper nada.
- Que sin especificar workspace, se use el default (backward compatible).

### Estado actual

Implementado casi completo. El request scope usa `X-Workspace`, el backend cae en `default` cuando falta el header, la consola envia el header siempre y los datos ya estan aislados por workspace. La operacion `DELETE /features/admin/workspaces/:key` hoy archiva por compatibilidad, no realiza borrado fisico ni bloquea explicitamente workspaces con datos.

### Criterios de Aceptacion

- [x] Se pueden crear multiples workspaces con identificadores unicos.
- [x] Features, segmentos, packs y tags son completamente aislados entre workspaces.
- [x] Un usuario puede tener roles diferentes en diferentes workspaces.
- [x] Las API keys de evaluacion solo acceden a features de su workspace.
- [x] La consola permite cambiar entre workspaces.
- [x] Feature keys pueden repetirse entre workspaces.
- [x] Los datos existentes migran al workspace "default" automaticamente.
- [ ] No se puede eliminar un workspace que contiene datos.
- [x] Sin indicar workspace, todo funciona como hoy (backward compatible).

### Deuda abierta

- Definir si se quiere mantener el modelo actual de archivado o agregar una validacion explicita de "workspace no eliminable si contiene datos".

---

## Orden de Implementacion Sugerido

El orden original ya no aplica como plan futuro. Se conserva como referencia historica de como se penso el rollout:

| Prioridad | Feature | Razon del orden | Estado real |
|-----------|---------|----------------|-------------|
| 1 | Percentage Rollouts | Es la base para canary releases y se necesita antes de A/B testing | Implementado |
| 2 | Historial de Cambios | Habilita auditoria para todas las features siguientes | Implementado |
| 3 | Scheduled Rollouts | Se beneficia del historial de cambios para registrar ejecuciones | Implementado parcial |
| 4 | Multi-Workspace | Cambio estructural que convenia hacer antes de A/B testing | Implementado |
| 5 | A/B Testing | Reutiliza bucketing de rollouts y requiere historial | Implementado |
