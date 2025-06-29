const monitorAmassEnumCompanyScanStatus = async (
  scanId, 
  setIsScanning, 
  setAmassEnumCompanyScans, 
  setMostRecentAmassEnumCompanyScan, 
  setMostRecentAmassEnumCompanyScanStatus,
  setAmassEnumCompanyCloudDomains = null
) => {
  let attempts = 0;
  const maxAttempts = 600; // 10 minutes with 1-second intervals
  
  const poll = async () => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-enum-company/status/${scanId}`
      );
      
      if (!response.ok) {
        throw new Error('Failed to fetch scan status');
      }
      
      const scanStatus = await response.json();
      setMostRecentAmassEnumCompanyScan(scanStatus);
      setMostRecentAmassEnumCompanyScanStatus(scanStatus.status);
      
      // Update scans list
      if (setAmassEnumCompanyScans) {
        setAmassEnumCompanyScans(prevScans => {
          const updatedScans = prevScans.map(scan => 
            scan.scan_id === scanId ? scanStatus : scan
          );
          
          // If scan not found in list, add it
          if (!updatedScans.find(scan => scan.scan_id === scanId)) {
            updatedScans.unshift(scanStatus);
          }
          
          return updatedScans;
        });
      }
      
      if (scanStatus.status === 'success') {
        setIsScanning(false);
        
        // Fetch cloud domains if scan completed successfully
        if (setAmassEnumCompanyCloudDomains) {
          try {
            const cloudDomainsResponse = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-enum-company/cloud-domains/${scanId}`
            );
            
            if (cloudDomainsResponse.ok) {
              const cloudDomains = await cloudDomainsResponse.json();
              setAmassEnumCompanyCloudDomains(cloudDomains);
            }
          } catch (error) {
            console.error('Error fetching cloud domains:', error);
          }
        }
        
        return scanStatus;
      } else if (scanStatus.status === 'failed' || scanStatus.status === 'error') {
        setIsScanning(false);
        console.error('Amass Enum Company scan failed:', scanStatus.error);
        return scanStatus;
      } else if (scanStatus.status === 'pending' || scanStatus.status === 'running') {
        attempts++;
        
        if (attempts >= maxAttempts) {
          console.error('Amass Enum Company scan monitoring timed out');
          setIsScanning(false);
          return scanStatus;
        }
        
        // Continue polling
        setTimeout(poll, 1000);
      }
    } catch (error) {
      console.error('Error monitoring Amass Enum Company scan:', error);
      attempts++;
      
      if (attempts >= maxAttempts) {
        setIsScanning(false);
        return null;
      }
      
      // Retry after error
      setTimeout(poll, 2000);
    }
  };
  
  // Start polling
  setTimeout(poll, 1000);
};

export default monitorAmassEnumCompanyScanStatus; 