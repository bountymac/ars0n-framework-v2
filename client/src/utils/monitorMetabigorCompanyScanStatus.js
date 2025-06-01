const monitorMetabigorCompanyScanStatus = async (
  activeTarget,
  setMetabigorCompanyScans,
  setMostRecentMetabigorCompanyScan,
  setIsMetabigorCompanyScanning,
  setMostRecentMetabigorCompanyScanStatus
) => {
  if (!activeTarget) return;

  try {
    console.log('[METABIGOR-COMPANY] Monitoring scan status for target:', activeTarget.id);
    
    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/metabigor-company`
    );

    if (!response.ok) {
      throw new Error('Failed to fetch Metabigor Company scans');
    }

    const scans = await response.json();
    if (!Array.isArray(scans)) {
      setMetabigorCompanyScans([]);
      setMostRecentMetabigorCompanyScan(null);
      setMostRecentMetabigorCompanyScanStatus(null);
      setIsMetabigorCompanyScanning(false);
      return;
    }

    console.log('[METABIGOR-COMPANY] Retrieved', scans.length, 'scans');
    setMetabigorCompanyScans(scans);

    if (scans.length > 0) {
      const mostRecentScan = scans.reduce((latest, scan) => {
        const scanDate = new Date(scan.created_at);
        return scanDate > new Date(latest.created_at) ? scan : latest;
      }, scans[0]);

      console.log('[METABIGOR-COMPANY] Most recent scan status:', mostRecentScan.status);
      setMostRecentMetabigorCompanyScan(mostRecentScan);
      setMostRecentMetabigorCompanyScanStatus(mostRecentScan.status);

      if (mostRecentScan.status === 'pending') {
        setIsMetabigorCompanyScanning(true);
        setTimeout(() => {
          monitorMetabigorCompanyScanStatus(
            activeTarget,
            setMetabigorCompanyScans,
            setMostRecentMetabigorCompanyScan,
            setIsMetabigorCompanyScanning,
            setMostRecentMetabigorCompanyScanStatus
          );
        }, 5000);
      } else {
        setIsMetabigorCompanyScanning(false);
      }
    } else {
      setMostRecentMetabigorCompanyScan(null);
      setMostRecentMetabigorCompanyScanStatus(null);
      setIsMetabigorCompanyScanning(false);
    }
  } catch (error) {
    console.error('[METABIGOR-COMPANY] Error monitoring scan status:', error);
    setIsMetabigorCompanyScanning(false);
    setMostRecentMetabigorCompanyScan(null);
    setMostRecentMetabigorCompanyScanStatus(null);
    setMetabigorCompanyScans([]);
  }
};

export default monitorMetabigorCompanyScanStatus; 