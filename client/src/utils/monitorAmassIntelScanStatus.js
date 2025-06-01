const monitorAmassIntelScanStatus = (
  activeTarget,
  setAmassIntelScans,
  setMostRecentAmassIntelScan,
  setIsAmassIntelScanning,
  setMostRecentAmassIntelScanStatus
) => {
  if (!activeTarget || !activeTarget.id) return;

  const intervalId = setInterval(async () => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/amass-intel`
      );
      if (!response.ok) {
        throw new Error('Failed to fetch Amass Intel scans');
      }
      const scans = await response.json();
      setAmassIntelScans(scans);

      if (scans && scans.length > 0) {
        const mostRecentScan = scans[0];
        setMostRecentAmassIntelScan(mostRecentScan);
        setMostRecentAmassIntelScanStatus(mostRecentScan.status);

        if (mostRecentScan.status === 'success' || mostRecentScan.status === 'error') {
          setIsAmassIntelScanning(false);
          clearInterval(intervalId);
        }
      }
    } catch (error) {
      console.error('Error monitoring Amass Intel scan status:', error);
      setIsAmassIntelScanning(false);
      clearInterval(intervalId);
    }
  }, 5000);

  return intervalId;
};

export default monitorAmassIntelScanStatus; 