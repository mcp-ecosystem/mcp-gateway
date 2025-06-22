import { Avatar } from '@heroui/react';
import { ProviderIcon as LobeProviderIcon } from '@lobehub/icons';
import React from 'react';

interface ProviderIconProps {
  providerId: string;
  name: string;
  size?: number;
  className?: string;
  fallbackUrl?: string;
}

const ProviderIcon: React.FC<ProviderIconProps> = ({ 
  providerId, 
  name, 
  size = 24,
  className = '',
  fallbackUrl 
}) => {
  try {
    // Use LobeHub's ProviderIcon directly, following their pattern
    return (
      <div className={`inline-flex items-center justify-center ${className}`}>
        <LobeProviderIcon
          provider={providerId}
          size={size}
          style={{ borderRadius: 6 }}
          type={'avatar'}
        />
      </div>
    );
  } catch (error) {
    console.warn(`Icon not found for provider: ${providerId}`, error);
    
    // Fallback to Avatar with either fallbackUrl or first letter
    return (
      <Avatar
        size={size <= 20 ? 'sm' : size <= 28 ? 'md' : 'lg'}
        src={fallbackUrl}
        className={className}
        name={name}
        fallback={name.charAt(0).toUpperCase()}
      />
    );
  }
};

export default ProviderIcon;