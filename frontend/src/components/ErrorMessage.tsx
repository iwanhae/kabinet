import React from 'react';
import { AlertCircle } from 'lucide-react';

interface ErrorMessageProps {
  message: string;
  onRetry?: () => void;
}

export const ErrorMessage: React.FC<ErrorMessageProps> = ({ message, onRetry }) => {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 max-w-md">
        <div className="flex items-center space-x-3">
          <AlertCircle className="w-6 h-6 text-red-600" />
          <div>
            <h3 className="text-sm font-medium text-red-800">오류가 발생했습니다</h3>
            <p className="text-sm text-red-700 mt-1">{message}</p>
            {onRetry && (
              <button
                onClick={onRetry}
                className="mt-3 btn-primary"
              >
                다시 시도
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};