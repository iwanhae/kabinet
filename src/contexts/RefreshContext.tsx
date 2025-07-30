import React, { createContext, useState, useContext, useMemo } from "react";

interface RefreshContextType {
  refreshKey: number;
  triggerRefresh: () => void;
}

const RefreshContext = createContext<RefreshContextType | undefined>(undefined);

export const RefreshProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [refreshKey, setRefreshKey] = useState(0);
  
  const triggerRefresh = () => {
    setRefreshKey((prevKey) => prevKey + 1);
  };

  const value = useMemo(() => ({ refreshKey, triggerRefresh }), [refreshKey]);

  return (
    <RefreshContext.Provider value={value}>{children}</RefreshContext.Provider>
  );
};

export const useRefresh = (): RefreshContextType => {
  const context = useContext(RefreshContext);
  if (!context) {
    throw new Error("useRefresh must be used within a RefreshProvider");
  }
  return context;
};
