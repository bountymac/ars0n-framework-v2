const fetchAmassIntelScans = async (
  activeTarget,
  setAmassIntelScans,
  setMostRecentAmassIntelScan,
  setMostRecentAmassIntelScanStatus
) => {
  if (!activeTarget || !activeTarget.id) return;

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
    }
  } catch (error) {
    console.error('Error fetching Amass Intel scans:', error);
  }
};

export default fetchAmassIntelScans; 