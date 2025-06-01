const monitorMetabigorCompanyScanStatus = (
  activeTarget,
  setScans,
  setMostRecentScan,
  setIsScanning,
  setMostRecentScanStatus
) => {
  const interval = setInterval(async () => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/metabigor-company`
      );
      if (!response.ok) {
        throw new Error('Failed to fetch Metabigor Company scans');
      }
      const scans = await response.json();
      if (scans && scans.length > 0) {
        const mostRecentScan = scans[0];
        setMostRecentScan(mostRecentScan);
        setMostRecentScanStatus(mostRecentScan.status);
        setScans(scans);

        if (mostRecentScan.status === 'success' || mostRecentScan.status === 'error') {
          setIsScanning(false);
          clearInterval(interval);
        }
      }
    } catch (error) {
      console.error('Error monitoring Metabigor Company scan status:', error);
      clearInterval(interval);
      setIsScanning(false);
    }
  }, 5000);

  return interval;
};

export default monitorMetabigorCompanyScanStatus; 