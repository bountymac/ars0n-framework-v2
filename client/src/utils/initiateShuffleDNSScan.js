const initiateShuffleDNSScan = async (
  activeTarget,
  monitorShuffleDNSScanStatus,
  setIsShuffleDNSScanning,
  setShuffleDNSScans,
  setMostRecentShuffleDNSScanStatus,
  setMostRecentShuffleDNSScan
) => {
  if (!activeTarget || !activeTarget.scope_target) {
    console.error('No active target or invalid target format');
    return;
  }

  const domain = activeTarget.scope_target.replace('*.', '');
  if (!domain) {
    console.error('Invalid domain');
    return;
  }

  try {
    setIsShuffleDNSScanning(true);
    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/shuffledns/run`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          fqdn: domain
        }),
      }
    );

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to initiate ShuffleDNS scan: ${errorText}`);
    }

    const data = await response.json();

    monitorShuffleDNSScanStatus(
      activeTarget,
      setShuffleDNSScans,
      setMostRecentShuffleDNSScan,
      setIsShuffleDNSScanning,
      setMostRecentShuffleDNSScanStatus
    );

    return data;
  } catch (error) {
    console.error('Error initiating ShuffleDNS scan:', error);
    setIsShuffleDNSScanning(false);
    setMostRecentShuffleDNSScan(null);
    setMostRecentShuffleDNSScanStatus(null);
  }
};

export default initiateShuffleDNSScan; 