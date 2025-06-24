const monitorMetabigorCompanyScanStatus = (
  activeTarget,
  setMetabigorCompanyScans,
  setMostRecentMetabigorCompanyScan,
  setIsMetabigorCompanyScanning,
  setMostRecentMetabigorCompanyScanStatus,
  setMetabigorNetworkRanges
) => {
  if (!activeTarget || !activeTarget.id) return;

  const intervalId = setInterval(async () => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/metabigor-company`
      );
      if (!response.ok) {
        throw new Error('Failed to fetch Metabigor Company scans');
      }
      const scans = await response.json();
      setMetabigorCompanyScans(scans);

      if (scans && scans.length > 0) {
        const mostRecentScan = scans[0];
        setMostRecentMetabigorCompanyScan(mostRecentScan);
        setMostRecentMetabigorCompanyScanStatus(mostRecentScan.status);

        // Fetch network ranges when scan is completed successfully
        if (mostRecentScan.status === 'success' && mostRecentScan.scan_id && setMetabigorNetworkRanges) {
          try {
            const networkRangesResponse = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/metabigor-company/${mostRecentScan.scan_id}/networks`
            );
            if (networkRangesResponse.ok) {
              const networkRanges = await networkRangesResponse.json();
              setMetabigorNetworkRanges(Array.isArray(networkRanges) ? networkRanges : []);
            }
          } catch (networkError) {
            console.error('Error fetching Metabigor network ranges:', networkError);
            setMetabigorNetworkRanges([]);
          }
        }

        if (mostRecentScan.status === 'success' || mostRecentScan.status === 'error') {
          setIsMetabigorCompanyScanning(false);
          clearInterval(intervalId);
        }
      }
    } catch (error) {
      console.error('Error monitoring Metabigor Company scan status:', error);
      setIsMetabigorCompanyScanning(false);
      clearInterval(intervalId);
    }
  }, 5000);

  return intervalId;
};

export default monitorMetabigorCompanyScanStatus; 