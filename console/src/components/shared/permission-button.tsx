import type { Permission } from '@/auth/roles';
import type { ButtonProps } from '@/components/ui/button';

import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { usePermissions } from '@/hooks/use-permissions';

interface PermissionButtonProps extends ButtonProps {
  permission: Permission;
  tooltipMessage?: string;
}

export function PermissionButton({
  permission,
  tooltipMessage = 'Sin permisos',
  ...props
}: PermissionButtonProps) {
  const { can } = usePermissions();
  const allowed = can(permission);

  if (!allowed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span tabIndex={0}>
            <Button {...props} disabled />
          </span>
        </TooltipTrigger>
        <TooltipContent>{tooltipMessage}</TooltipContent>
      </Tooltip>
    );
  }

  return <Button {...props} />;
}
